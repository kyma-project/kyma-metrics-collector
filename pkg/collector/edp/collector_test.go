package edp

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/kyma-project/kyma-metrics-collector/pkg/collector"
	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
	"net/http"
	"testing"

	"context"
	"encoding/json"
	"github.com/kyma-project/kyma-metrics-collector/pkg/collector/edp/stubs"
	"github.com/kyma-project/kyma-metrics-collector/pkg/logger"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
	kmctesting "github.com/kyma-project/kyma-metrics-collector/pkg/testing"
	"github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
	"io"
	"k8s.io/utils/ptr"
	"strconv"
)

const (
	testDataStream = "dataStream"
	retryCount     = 1
	backendName    = "edp"
)

func TestCollector_CollectAndSend(t *testing.T) {
	scannerID1 := resource.ScannerID("scanner1")
	scannerID2 := resource.ScannerID("scanner2")

	// In order to test the aggregation of EDP measurements, we are going to executes the tests with two scanners: scanner1 and scanner2.
	// scanner1 will behave the same in all test cases: successful scan and successful conversion to EDP measurement.
	// scanner2 will behave differently in each test case:
	// case 1: scanner2 succeeds in scanning and conversion to EDP measurement is successful.
	// case 2: scanner2 fails in scanning, but previous scan exists. So, the previous scan will be used for conversion to EDP measurement.
	// case 3: scanner2 fails in scanning and previous scan doesn't exist. So, nothing else we can do here and payload will be sent to EDP without scanner2's data.
	// case 4: scanner2 succeeds in scanning, but conversion to EDP measurement fails. So, the previous scan will be used for conversion to EDP measurement.
	// case 5: scanner2 succeeds in scanning, but conversion to EDP measurement fails and previous scan doesn't exist. So, nothing else we can do here and payload will be sent to EDP without scanner2's data.
	// case 6: scanner2 succeeds in scanning, but conversion to EDP measurement fails and previous scan exists, but conversion of previous scan to EDP measurement also fails. So, nothing else we can do here and payload will be sent to EDP without scanner2's data.
	testCases := []struct {
		name string

		scanError2      error
		EDPMeasurement2 resource.EDPMeasurement
		EDPError2       error

		previousScanMap collector.ScanMap

		expectedAggregatedEDPMeasurement resource.EDPMeasurement

		// expectedToUpdateScanner2InNewScanMap determines what should be the expected value of the scanner2 in the new scan map.
		// If it is nil, scanner2 should not be in the new scan map.
		// If it is true, scanner2 should have the new scan value in the new scan map.
		// If it is false, scanner2 should have the previous scan in the new scan map.
		expectedToUpdateScanner2InNewScanMap *bool

		expectedErrInCollectAndSend bool

		expectedScanConversionToSucceed2 bool
	}{
		{
			name: "scanner2 succeeds in scanning and conversion to EDP measurement is successful",

			scanError2: nil,
			EDPMeasurement2: resource.EDPMeasurement{
				VMTypes: []resource.VMType{
					{Name: "m5.large", Count: 1},
				},
				ProvisionedCPUs:  2,
				ProvisionedRAMGb: 2,
				ProvisionedVolumes: resource.ProvisionedVolumes{
					SizeGbTotal:   2,
					Count:         2,
					SizeGbRounded: 2,
				},
			},
			EDPError2: nil,

			previousScanMap: nil,

			expectedAggregatedEDPMeasurement: resource.EDPMeasurement{
				VMTypes: []resource.VMType{
					{Name: "t2.micro", Count: 1},
					{Name: "m5.large", Count: 1},
				},
				ProvisionedCPUs:  3,
				ProvisionedRAMGb: 3,
				ProvisionedVolumes: resource.ProvisionedVolumes{
					SizeGbTotal:   3,
					Count:         3,
					SizeGbRounded: 3,
				},
			},

			expectedToUpdateScanner2InNewScanMap: ptr.To(true),

			expectedErrInCollectAndSend: false,

			expectedScanConversionToSucceed2: true,
		},
		{
			name: "scanner2 fails in scanning, but previous scan exists. So, the previous scan will be used for conversion to EDP measurement",

			scanError2:      fmt.Errorf("failed to scan"),
			EDPMeasurement2: resource.EDPMeasurement{},
			EDPError2:       nil,

			previousScanMap: collector.ScanMap{
				scannerID2: stubs.NewScan(resource.EDPMeasurement{
					VMTypes: []resource.VMType{
						{Name: "m5.large", Count: 1},
					},
					ProvisionedCPUs:  2,
					ProvisionedRAMGb: 2,
					ProvisionedVolumes: resource.ProvisionedVolumes{
						SizeGbTotal:   2,
						Count:         2,
						SizeGbRounded: 2,
					},
				}, nil),
			},

			expectedAggregatedEDPMeasurement: resource.EDPMeasurement{
				VMTypes: []resource.VMType{
					{Name: "t2.micro", Count: 1},
					{Name: "m5.large", Count: 1},
				},
				ProvisionedCPUs:  3,
				ProvisionedRAMGb: 3,
				ProvisionedVolumes: resource.ProvisionedVolumes{
					SizeGbTotal:   3,
					Count:         3,
					SizeGbRounded: 3,
				},
			},

			expectedToUpdateScanner2InNewScanMap: ptr.To(false),

			expectedErrInCollectAndSend: true,

			expectedScanConversionToSucceed2: true,
		},
		{
			name: "scanner2 fails in scanning and previous scan doesn't exist. So, nothing else we can do here and payload will be sent to EDP without scanner2's data",

			scanError2:      fmt.Errorf("failed to scan"),
			EDPMeasurement2: resource.EDPMeasurement{},
			EDPError2:       nil,

			previousScanMap: collector.ScanMap{},

			expectedAggregatedEDPMeasurement: resource.EDPMeasurement{
				VMTypes: []resource.VMType{
					{Name: "t2.micro", Count: 1},
				},
				ProvisionedCPUs:  1,
				ProvisionedRAMGb: 1,
				ProvisionedVolumes: resource.ProvisionedVolumes{
					SizeGbTotal:   1,
					Count:         1,
					SizeGbRounded: 1,
				},
			},

			expectedToUpdateScanner2InNewScanMap: nil,

			expectedErrInCollectAndSend: true,

			expectedScanConversionToSucceed2: false,
		},
		{
			name: "scanner2 succeeds in scanning, but conversion to EDP measurement fails. So, the previous scan will be used for conversion to EDP measurement",

			scanError2:      nil,
			EDPMeasurement2: resource.EDPMeasurement{},
			EDPError2:       fmt.Errorf("failed to convert scan to EDP measurement"),

			previousScanMap: collector.ScanMap{
				scannerID2: stubs.NewScan(resource.EDPMeasurement{
					VMTypes: []resource.VMType{
						{Name: "m5.large", Count: 1},
					},
					ProvisionedCPUs:  2,
					ProvisionedRAMGb: 2,
					ProvisionedVolumes: resource.ProvisionedVolumes{
						SizeGbTotal:   2,
						Count:         2,
						SizeGbRounded: 2,
					},
				}, nil),
			},

			expectedAggregatedEDPMeasurement: resource.EDPMeasurement{
				VMTypes: []resource.VMType{
					{Name: "t2.micro", Count: 1},
					{Name: "m5.large", Count: 1},
				},
				ProvisionedCPUs:  3,
				ProvisionedRAMGb: 3,
				ProvisionedVolumes: resource.ProvisionedVolumes{
					SizeGbTotal:   3,
					Count:         3,
					SizeGbRounded: 3,
				},
			},

			expectedToUpdateScanner2InNewScanMap: ptr.To(false),

			expectedErrInCollectAndSend: true,

			expectedScanConversionToSucceed2: true,
		},
		{
			name: "scanner2 succeeds in scanning, but conversion to EDP measurement fails and previous scan doesn't exist. So, nothing else we can do here and payload will be sent to EDP without scanner2's data",

			scanError2:      nil,
			EDPMeasurement2: resource.EDPMeasurement{},
			EDPError2:       fmt.Errorf("failed to convert scan to EDP measurement"),

			previousScanMap: collector.ScanMap{},

			expectedAggregatedEDPMeasurement: resource.EDPMeasurement{
				VMTypes: []resource.VMType{
					{Name: "t2.micro", Count: 1},
				},
				ProvisionedCPUs:  1,
				ProvisionedRAMGb: 1,
				ProvisionedVolumes: resource.ProvisionedVolumes{
					SizeGbTotal:   1,
					Count:         1,
					SizeGbRounded: 1,
				},
			},

			expectedToUpdateScanner2InNewScanMap: ptr.To(true),

			expectedErrInCollectAndSend: true,

			expectedScanConversionToSucceed2: false,
		},
		{
			name: "scanner2 succeeds in scanning, but conversion to EDP measurement fails and previous scan exists, but conversion of previous scan to EDP measurement also fails. So, nothing else we can do here and payload will be sent to EDP without scanner2's data",

			scanError2:      nil,
			EDPMeasurement2: resource.EDPMeasurement{},
			EDPError2:       fmt.Errorf("failed to convert scan to EDP measurement"),

			previousScanMap: collector.ScanMap{
				scannerID2: stubs.NewScan(resource.EDPMeasurement{}, fmt.Errorf("failed to convert scan to EDP measurement")),
			},

			expectedAggregatedEDPMeasurement: resource.EDPMeasurement{
				VMTypes: []resource.VMType{
					{Name: "t2.micro", Count: 1},
				},
				ProvisionedCPUs:  1,
				ProvisionedRAMGb: 1,
				ProvisionedVolumes: resource.ProvisionedVolumes{
					SizeGbTotal:   1,
					Count:         1,
					SizeGbRounded: 1,
				},
			},

			expectedToUpdateScanner2InNewScanMap: ptr.To(true),

			expectedErrInCollectAndSend: true,

			expectedScanConversionToSucceed2: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			collector.TotalScans.Reset()
			collector.TotalScansConverted.Reset()

			instanceID := uuid.New().String()
			runtimeID := uuid.New().String()
			subAccountID := uuid.New().String()
			globalAccountID := uuid.New().String()
			shootName := uuid.New().String()

			EDPMeasurement1 := resource.EDPMeasurement{
				VMTypes: []resource.VMType{
					{Name: "t2.micro", Count: 1},
				},
				ProvisionedCPUs:  1,
				ProvisionedRAMGb: 1,
				ProvisionedVolumes: resource.ProvisionedVolumes{
					SizeGbTotal:   1,
					Count:         1,
					SizeGbRounded: 1,
				},
			}

			scan1 := stubs.NewScan(EDPMeasurement1, nil)
			scanner1 := stubs.NewScanner(scan1, nil, scannerID1)

			scan2 := stubs.NewScan(tc.EDPMeasurement2, tc.EDPError2)
			scanner2 := stubs.NewScanner(scan2, tc.scanError2, scannerID2)

			expectedNewScanMap := collector.ScanMap{
				scannerID1: scan1,
			}

			if tc.expectedToUpdateScanner2InNewScanMap != nil {
				if *tc.expectedToUpdateScanner2InNewScanMap {
					expectedNewScanMap[scannerID2] = scan2
				} else {
					expectedNewScanMap[scannerID2] = tc.previousScanMap[scannerID2]
				}
			}

			expectedPath := fmt.Sprintf("/namespaces/%s/dataStreams/%s/%s/dataTenants/%s/%s/events", testNamespace, testDataStream, testDataStreamVersion, subAccountID, testEnv)
			expectedHeaders := expectedHeadersInEDPReq()
			edpPayloadSent := false
			edpTestHandler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				edpPayloadSent = true

				require.Equal(t, expectedHeaders, req.Header)
				require.Equal(t, expectedPath, req.URL.Path)
				require.Equal(t, http.MethodPost, req.Method)

				// Read the request body
				body, err := io.ReadAll(req.Body)
				if err != nil {
					http.Error(rw, "Failed to read request body", http.StatusInternalServerError)
					return
				}
				defer req.Body.Close()

				// Parse the JSON body into the Payload struct
				var payload Payload
				err = json.Unmarshal(body, &payload)
				if err != nil {
					http.Error(rw, "Failed to parse JSON", http.StatusBadRequest)
					return
				}
				require.Equal(t, tc.expectedAggregatedEDPMeasurement, payload.Compute)

				rw.WriteHeader(http.StatusCreated)
			})

			srv := kmctesting.StartTestServer(expectedPath, edpTestHandler, gomega.NewGomegaWithT(t))
			defer srv.Close()

			edpConfig := newEDPConfig(srv.URL)
			edpClient := NewClient(edpConfig, logger.NewLogger(zapcore.DebugLevel))

			EDPCollector := NewCollector(edpClient, scanner1, scanner2)

			runtimeInfo := runtime.Info{
				InstanceID:      instanceID,
				RuntimeID:       runtimeID,
				SubAccountID:    subAccountID,
				GlobalAccountID: globalAccountID,
				ShootName:       shootName,
			}

			scanMap, err := EDPCollector.CollectAndSend(context.Background(), &runtimeInfo, tc.previousScanMap)
			if tc.expectedErrInCollectAndSend {
				require.NotNil(t, err)
			} else {
				require.Nil(t, err)
			}
			require.True(t, edpPayloadSent)
			require.Equal(t, expectedNewScanMap, scanMap)

			// check prometheus metrics.
			// metrics: totalScans for scanner1
			gotMetrics, err := collector.TotalScans.GetMetricWithLabelValues(
				strconv.FormatBool(true),
				string(scannerID1),
				runtimeInfo.ShootName,
				runtimeInfo.InstanceID,
				runtimeInfo.RuntimeID,
				runtimeInfo.SubAccountID,
				runtimeInfo.GlobalAccountID,
			)
			require.Nil(t, err)
			require.Equal(t, float64(1), testutil.ToFloat64(gotMetrics))

			// metrics: totalScans for scanner2
			gotMetrics, err = collector.TotalScans.GetMetricWithLabelValues(
				strconv.FormatBool(tc.scanError2 == nil),
				string(scannerID2),
				runtimeInfo.ShootName,
				runtimeInfo.InstanceID,
				runtimeInfo.RuntimeID,
				runtimeInfo.SubAccountID,
				runtimeInfo.GlobalAccountID,
			)
			require.Nil(t, err)
			require.Equal(t, float64(1), testutil.ToFloat64(gotMetrics))

			// metrics: TotalScansConverted for scanner1
			gotMetrics, err = collector.TotalScansConverted.GetMetricWithLabelValues(
				strconv.FormatBool(true),
				string(scannerID1),
				backendName,
				runtimeInfo.ShootName,
				runtimeInfo.InstanceID,
				runtimeInfo.RuntimeID,
				runtimeInfo.SubAccountID,
				runtimeInfo.GlobalAccountID,
			)
			require.Nil(t, err)
			require.Equal(t, float64(1), testutil.ToFloat64(gotMetrics))

			// metrics: TotalScansConverted for scanner2
			gotMetrics, err = collector.TotalScansConverted.GetMetricWithLabelValues(
				strconv.FormatBool(tc.expectedScanConversionToSucceed2),
				string(scannerID2),
				backendName,
				runtimeInfo.ShootName,
				runtimeInfo.InstanceID,
				runtimeInfo.RuntimeID,
				runtimeInfo.SubAccountID,
				runtimeInfo.GlobalAccountID,
			)
			require.Nil(t, err)
			require.Equal(t, float64(1), testutil.ToFloat64(gotMetrics))
		})
	}
}

func expectedHeadersInEDPReq() http.Header {
	return http.Header{
		"Authorization":   []string{fmt.Sprintf("Bearer %s", testToken)},
		"Accept-Encoding": []string{"gzip"},
		"User-Agent":      []string{"kyma-metrics-collector"},
		"Content-Type":    []string{"application/json;charset=utf-8"},
	}
}

func newEDPConfig(url string) *Config {
	return &Config{
		URL:               url,
		Token:             testToken,
		Namespace:         testNamespace,
		DataStreamName:    testDataStream,
		DataStreamVersion: testDataStreamVersion,
		DataStreamEnv:     testEnv,
		Timeout:           timeout,
		EventRetry:        retryCount,
	}
}
