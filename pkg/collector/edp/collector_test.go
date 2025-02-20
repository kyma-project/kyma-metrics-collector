package edp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
	"k8s.io/utils/ptr"

	"github.com/kyma-project/kyma-metrics-collector/pkg/collector"
	"github.com/kyma-project/kyma-metrics-collector/pkg/collector/edp/stubs"
	"github.com/kyma-project/kyma-metrics-collector/pkg/logger"
	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
	runtimestubs "github.com/kyma-project/kyma-metrics-collector/pkg/runtime/stubs"
	kmctesting "github.com/kyma-project/kyma-metrics-collector/pkg/testing"
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
	// case 1: scanner2 succeeds in scanning and conversion to EDP measurement succeeds.
	// case 2: scanner2 fails in scanning, but previous scan exists. So, the previous scan will be used for conversion to EDP measurement.
	// case 3: scanner2 fails in scanning and previous scan doesn't exist. So, nothing else we can do here and payload will be sent to EDP without scanner2's data.
	// case 4: scanner2 succeeds in scanning, but conversion to EDP measurement fails. So, the previous scan will be used for conversion to EDP measurement.
	// case 5: scanner2 succeeds in scanning, but conversion to EDP measurement fails and previous scan doesn't exist. So, nothing else we can do here and payload will be sent to EDP without scanner2's data.
	testCases := []struct {
		name string

		// error returned when scanner2 executes the scan
		scanError2 error
		// EDP measurement for scanner2
		EDPMeasurement2 resource.EDPMeasurement
		// error returned when the scan of scanner2 is converted to EDP measurement
		EDPError2 error

		previousScanMap collector.ScanMap

		// expected aggregated EDP measurement that should be sent to EDP backend
		expectedAggregatedEDPMeasurement resource.EDPMeasurement

		// expectedToUpdateScanner2InNewScanMap determines what should be the expected value of the scanner2 in the new scan map.
		// If it is nil, scanner2 should not be in the new scan map.
		// If it is true, scanner2 should have the new scan value in the new scan map.
		// If it is false, scanner2 should have the previous scan in the new scan map.
		expectedToUpdateScanner2InNewScanMap *bool

		// expected error returned by CollectAndSend func
		expectedErrInCollectAndSend bool

		// expected "success" label value in totalScansConverted prometheus metric for scanner2
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
					{Name: "m5.large", Count: 1},
					{Name: "t2.micro", Count: 1},
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
					{Name: "m5.large", Count: 1},
					{Name: "t2.micro", Count: 1},
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
					{Name: "m5.large", Count: 1},
					{Name: "t2.micro", Count: 1},
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

			expectedScanConversionToSucceed2: false,
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

			expectedToUpdateScanner2InNewScanMap: nil,

			expectedErrInCollectAndSend: true,

			expectedScanConversionToSucceed2: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := gomega.NewGomegaWithT(t)

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

				g.Expect(req.Header).To(gomega.Equal(expectedHeaders))
				g.Expect(req.URL.Path).To(gomega.Equal(expectedPath))
				g.Expect(req.Method).To(gomega.Equal(http.MethodPost))

				// Read the request body
				body, err := io.ReadAll(req.Body)
				if err != nil {
					http.Error(rw, "Failed to read request body", http.StatusInternalServerError)
					return
				}
				defer req.Body.Close()

				// Parse the JSON body into the payload struct
				var payload payload

				err = json.Unmarshal(body, &payload)
				if err != nil {
					http.Error(rw, "Failed to parse JSON", http.StatusBadRequest)
					return
				}

				sort.Slice(payload.Compute.VMTypes, func(i, j int) bool {
					return payload.Compute.VMTypes[i].Name < payload.Compute.VMTypes[j].Name
				})
				g.Expect(payload.Compute).To(gomega.Equal(tc.expectedAggregatedEDPMeasurement))

				rw.WriteHeader(http.StatusCreated)
			})

			srv := kmctesting.StartTestServer(expectedPath, edpTestHandler, g)
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
			clients := runtimestubs.Clients{}

			scanMap, err := EDPCollector.CollectAndSend(t.Context(), &runtimeInfo, clients, tc.previousScanMap)
			if tc.expectedErrInCollectAndSend {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.True(t, edpPayloadSent)
			require.Equal(t, expectedNewScanMap, scanMap)

			// check prometheus metrics.
			checkPrometheusMetrics(t, scannerID1, scannerID2, runtimeInfo, tc.scanError2, tc.expectedScanConversionToSucceed2)
		})
	}
}

func checkPrometheusMetrics(t *testing.T, scannerID1, scannerID2 resource.ScannerID, runtimeInfo runtime.Info, scanError2 error, expectedScanConversionToSucceed2 bool) {
	t.Helper()

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
	require.NoError(t, err)
	require.InEpsilon(t, float64(1), testutil.ToFloat64(gotMetrics), kmctesting.Delta)

	// metrics: totalScans for scanner2
	gotMetrics, err = collector.TotalScans.GetMetricWithLabelValues(
		strconv.FormatBool(scanError2 == nil),
		string(scannerID2),
		runtimeInfo.ShootName,
		runtimeInfo.InstanceID,
		runtimeInfo.RuntimeID,
		runtimeInfo.SubAccountID,
		runtimeInfo.GlobalAccountID,
	)
	require.NoError(t, err)
	require.InEpsilon(t, float64(1), testutil.ToFloat64(gotMetrics), kmctesting.Delta)

	// metrics: totalScansConverted for scanner1
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
	require.NoError(t, err)
	require.InEpsilon(t, float64(1), testutil.ToFloat64(gotMetrics), kmctesting.Delta)

	// metrics: totalScansConverted for scanner2
	gotMetrics, err = collector.TotalScansConverted.GetMetricWithLabelValues(
		strconv.FormatBool(expectedScanConversionToSucceed2),
		string(scannerID2),
		backendName,
		runtimeInfo.ShootName,
		runtimeInfo.InstanceID,
		runtimeInfo.RuntimeID,
		runtimeInfo.SubAccountID,
		runtimeInfo.GlobalAccountID,
	)
	require.NoError(t, err)
	require.InEpsilon(t, float64(1), testutil.ToFloat64(gotMetrics), kmctesting.Delta)
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
