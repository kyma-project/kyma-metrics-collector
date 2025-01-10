package process

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	kebruntime "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/onsi/gomega"
	gocache "github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"go.uber.org/zap/zapcore"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/util/workqueue"

	kmccache "github.com/kyma-project/kyma-metrics-collector/pkg/cache"
	"github.com/kyma-project/kyma-metrics-collector/pkg/collector"
	"github.com/kyma-project/kyma-metrics-collector/pkg/config"
	kmckeb "github.com/kyma-project/kyma-metrics-collector/pkg/keb"
	"github.com/kyma-project/kyma-metrics-collector/pkg/logger"
	"github.com/kyma-project/kyma-metrics-collector/pkg/process/stubs"
	runtime2 "github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
	kmctesting "github.com/kyma-project/kyma-metrics-collector/pkg/testing"
)

const (
	// General.
	timeout    = 5 * time.Second
	bigTimeout = 10 * time.Second

	// KEB related variables.
	kebRuntimeResponseFilePath = "../testing/fixtures/runtimes_response_process.json"
	expectedPathPrefix         = "/runtimes"

	fecthedClustersMetricName = "kmc_process_fetched_clusters_total"
)

func TestPollKEBForRuntimes(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	t.Run("execute KEB poller for 2 times", func(t *testing.T) {
		runtimesResponse, err := kmctesting.LoadFixtureFromFile(kebRuntimeResponseFilePath)
		g.Expect(err).Should(gomega.BeNil())

		expectedRuntimes := new(kebruntime.RuntimesPage)
		err = json.Unmarshal(runtimesResponse, expectedRuntimes)
		g.Expect(err).Should(gomega.BeNil())

		timesVisited := 0
		expectedTimesVisited := 2

		var newProcess *Process

		getRuntimesHandler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			timesVisited += 1
			t.Logf("time visited: %d", timesVisited)
			g.Expect(req.URL.Path).To(gomega.Equal(expectedPathPrefix))

			_, err := rw.Write(runtimesResponse)
			g.Expect(err).Should(gomega.BeNil())
			rw.WriteHeader(http.StatusOK)
		})

		// Start a local test HTTP server
		srv := kmctesting.StartTestServer(expectedPathPrefix, getRuntimesHandler, g)
		defer srv.Close()
		// Wait until test server is ready
		g.Eventually(func() int {
			// Ignoring error is ok as it goes for retry for non-200 cases
			healthResp, err := http.Get(fmt.Sprintf("%s/health", srv.URL))
			t.Logf("retrying :%v", err)
			return healthResp.StatusCode
		}, timeout).Should(gomega.Equal(http.StatusOK))

		kebURL := fmt.Sprintf("%s%s", srv.URL, expectedPathPrefix)

		config := &kmckeb.Config{
			URL:              kebURL,
			Timeout:          timeout,
			RetryCount:       1,
			PollWaitDuration: 2 * time.Second,
		}
		kebClient := &kmckeb.Client{
			HTTPClient: http.DefaultClient,
			Logger:     logger.NewLogger(zapcore.InfoLevel),
			Config:     config,
		}

		queue := workqueue.NewTypedDelayingQueue[string]()
		cache := gocache.New(gocache.NoExpiration, gocache.NoExpiration)
		newProcess = &Process{
			KEBClient:      kebClient,
			Queue:          queue,
			Cache:          cache,
			ScrapeInterval: 0,
			Logger:         logger.NewLogger(zapcore.InfoLevel),
		}

		// Reset the cluster count necessary for clean slate of next tests
		kebFetchedClusters.Reset()

		go func() {
			newProcess.pollKEBForRuntimes()
		}()
		g.Eventually(func() int {
			return timesVisited
		}, 10*time.Second).Should(gomega.Equal(expectedTimesVisited))

		// Ensure metric exists
		metricName := fecthedClustersMetricName
		numberOfAllClusters := 4
		expectedMetricValue := 1

		g.Eventually(testutil.CollectAndCount(kebFetchedClusters, metricName)).Should(gomega.Equal(numberOfAllClusters))
		// check each metric with labels has the expected value
		for _, runtimeData := range expectedRuntimes.Data {
			verifyKEBAllClustersCountMetricValue(expectedMetricValue, g, runtimeData)
		}
	})
}

func TestIsProvisionedStatus(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	// t.Parallel()

	// const used in all test cases
	subAccountID := "c7db696a-32fa-48ee-9009-aa3e0034121e"
	shootName := "shoot-gKtxg"

	// test cases
	testCases := []struct {
		name         string
		givenRuntime kebruntime.RuntimeDTO
		expectedBool bool
	}{
		{
			name:         "should return true when runtime is in provisioning state succeeded and provisioning status is not nil and deprovisioning is nil",
			givenRuntime: kmctesting.NewRuntimesDTO(subAccountID, shootName, kmctesting.WithProvisioningSucceededStatus(kebruntime.StateSucceeded)),
			expectedBool: true,
		},
		{
			name:         "should return false when runtime is in provisioning state succeeded and deprovisioning is not nil",
			givenRuntime: kmctesting.NewRuntimesDTO(subAccountID, shootName, kmctesting.WithProvisionedAndDeprovisionedStatus(kebruntime.StateSucceeded)),
			expectedBool: false,
		},
		{
			name:         "should return false when runtime is in provisioning state failed and provisioning status is not nil and deprovisioning is nil",
			givenRuntime: kmctesting.NewRuntimesDTO(subAccountID, shootName, kmctesting.WithProvisioningFailedState),
			expectedBool: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			isProvisioned := isProvisionedStatus(tc.givenRuntime)

			// then
			g.Expect(isProvisioned).To(gomega.Equal(tc.expectedBool))
		})
	}
}

func TestIsTrackableState(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	// t.Parallel()

	// test cases
	testCases := []struct {
		name              string
		givenRuntimeState kebruntime.State
		expectedBool      bool
	}{
		{
			name:              "should return true when shoot is in succeeded state",
			givenRuntimeState: kebruntime.StateSucceeded,
			expectedBool:      true,
		},
		{
			name:              "should return true when shoot is in error state",
			givenRuntimeState: kebruntime.StateError,
			expectedBool:      true,
		},
		{
			name:              "should return true when shoot is in upgrading state",
			givenRuntimeState: kebruntime.StateUpgrading,
			expectedBool:      true,
		},
		{
			name:              "should return true when shoot is in updating state",
			givenRuntimeState: kebruntime.StateUpdating,
			expectedBool:      true,
		},
		{
			name:              "should return false when shoot is in deprovisioned state",
			givenRuntimeState: kebruntime.StateDeprovisioned,
			expectedBool:      false,
		},
		{
			name:              "should return false when shoot is in deprovisioned incomplete state",
			givenRuntimeState: kebruntime.StateDeprovisionIncomplete,
			expectedBool:      false,
		},
		{
			name:              "should return false when shoot is in deprovisioning  state",
			givenRuntimeState: kebruntime.StateDeprovisioning,
			expectedBool:      false,
		},
		{
			name:              "should return false when shoot is in failed state",
			givenRuntimeState: kebruntime.StateFailed,
			expectedBool:      false,
		},
		{
			name:              "should return false when shoot is in suspended state",
			givenRuntimeState: kebruntime.StateSuspended,
			expectedBool:      false,
		},
		{
			name:              "should return false when shoot is in provisioning state",
			givenRuntimeState: kebruntime.StateProvisioning,
			expectedBool:      false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			isTrackable := isTrackableState(tc.givenRuntimeState)

			// then
			g.Expect(isTrackable).To(gomega.Equal(tc.expectedBool))
		})
	}
}

func TestIsRuntimeTrackable(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	// t.Parallel()

	// const used in all test cases
	subAccountID := "c7db696a-32fa-48ee-9009-aa3e0034121e"
	shootName := "shoot-gKtxg"

	// test cases
	testCases := []struct {
		name         string
		givenRuntime kebruntime.RuntimeDTO
		expectedBool bool
	}{
		{
			name:         "should return true when runtime is in trackable state and provisioned status",
			givenRuntime: kmctesting.NewRuntimesDTO(subAccountID, shootName, kmctesting.WithProvisioningSucceededStatus(kebruntime.StateSucceeded)),
			expectedBool: true,
		},
		{
			name:         "should return true when runtime is in trackable state and deprovisioned status",
			givenRuntime: kmctesting.NewRuntimesDTO(subAccountID, shootName, kmctesting.WithProvisionedAndDeprovisionedStatus(kebruntime.StateSucceeded)),
			expectedBool: true,
		},
		{
			name:         "should return false when runtime is in non-trackable state and provisioned status",
			givenRuntime: kmctesting.NewRuntimesDTO(subAccountID, shootName, kmctesting.WithProvisioningSucceededStatus(kebruntime.StateDeprovisioning)),
			expectedBool: false,
		},
		{
			name:         "should return false when runtime is in non-trackable state and deprovisioned status",
			givenRuntime: kmctesting.NewRuntimesDTO(subAccountID, shootName, kmctesting.WithProvisionedAndDeprovisionedStatus(kebruntime.StateDeprovisioning)),
			expectedBool: false,
		},
		{
			name:         "should return false when runtime state has status deprovisioning",
			givenRuntime: kmctesting.NewRuntimesDTO(subAccountID, shootName, kmctesting.WithState(kebruntime.StateDeprovisioning)),
			expectedBool: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			isRuntimeTrackable := isRuntimeTrackable(tc.givenRuntime)

			// then
			g.Expect(isRuntimeTrackable).To(gomega.Equal(tc.expectedBool))
		})
	}
}

func TestPopulateCacheAndQueue(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	t.Run("runtimes with only provisioned status and other statuses with failures", func(t *testing.T) {
		// Reset the cluster count necessary for clean slate of next tests
		kebFetchedClusters.Reset()

		provisionedSuccessfullySubAccIDs := []string{uuid.New().String(), uuid.New().String()}
		provisionedFailedSubAccIDs := []string{uuid.New().String(), uuid.New().String()}
		cache := gocache.New(gocache.NoExpiration, gocache.NoExpiration)
		queue := workqueue.NewTypedDelayingQueue[string]()
		p := Process{
			Queue:  queue,
			Cache:  cache,
			Logger: logger.NewLogger(zapcore.InfoLevel),
		}
		runtimesPage := new(kebruntime.RuntimesPage)

		expectedQueue := workqueue.NewTypedDelayingQueue[string]()
		expectedCache := gocache.New(gocache.NoExpiration, gocache.NoExpiration)

		runtimesPage, expectedCache, expectedQueue, err := AddSuccessfulIDsToCacheQueueAndRuntimes(runtimesPage, provisionedSuccessfullySubAccIDs, expectedCache, expectedQueue)
		g.Expect(err).Should(gomega.BeNil())

		for _, failedID := range provisionedFailedSubAccIDs {
			shootName := fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5))
			runtime := kmctesting.NewRuntimesDTO(failedID, shootName, kmctesting.WithProvisioningFailedState)
			runtimesPage.Data = append(runtimesPage.Data, runtime)
		}

		p.populateCacheAndQueue(runtimesPage)
		g.Expect(*p.Cache).To(gomega.Equal(*expectedCache))
		g.Expect(areQueuesEqual(p.Queue, expectedQueue)).To(gomega.BeTrue())

		// Ensure metric exists
		metricName := fecthedClustersMetricName
		numberOfAllClusters := 4
		expectedMetricValue := 1

		g.Eventually(testutil.CollectAndCount(kebFetchedClusters, metricName)).Should(gomega.Equal(numberOfAllClusters))

		for _, runtimeData := range runtimesPage.Data {
			verifyKEBAllClustersCountMetricValue(expectedMetricValue, g, runtimeData)
		}
	})

	t.Run("runtimes with both provisioned and deprovisioned status", func(t *testing.T) {
		// Reset the cluster count necessary for clean slate of next tests
		kebFetchedClusters.Reset()

		provisionedSuccessfullySubAccIDs := []string{uuid.New().String(), uuid.New().String()}
		provisionedAndDeprovisionedSubAccIDs := []string{uuid.New().String(), uuid.New().String()}
		cache := gocache.New(gocache.NoExpiration, gocache.NoExpiration)
		queue := workqueue.NewTypedDelayingQueue[string]()
		p := Process{
			Queue:  queue,
			Cache:  cache,
			Logger: logger.NewLogger(zapcore.InfoLevel),
		}
		runtimesPage := new(kebruntime.RuntimesPage)

		expectedQueue := workqueue.NewTypedDelayingQueue[string]()
		expectedCache := gocache.New(gocache.NoExpiration, gocache.NoExpiration)

		runtimesPage, expectedCache, expectedQueue, err := AddSuccessfulIDsToCacheQueueAndRuntimes(runtimesPage, provisionedSuccessfullySubAccIDs, expectedCache, expectedQueue)
		g.Expect(err).Should(gomega.BeNil())

		for _, failedID := range provisionedAndDeprovisionedSubAccIDs {
			rntme := kmctesting.NewRuntimesDTO(failedID, fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5)), kmctesting.WithProvisionedAndDeprovisionedStatus(kebruntime.StateDeprovisioned))
			runtimesPage.Data = append(runtimesPage.Data, rntme)
		}

		p.populateCacheAndQueue(runtimesPage)
		g.Expect(*p.Cache).To(gomega.Equal(*expectedCache))
		g.Expect(areQueuesEqual(p.Queue, expectedQueue)).To(gomega.BeTrue())

		// Ensure metric exists
		metricName := fecthedClustersMetricName
		numberOfAllClusters := 4
		expectedMetricValue := 1

		g.Eventually(testutil.CollectAndCount(kebFetchedClusters, metricName)).Should(gomega.Equal(numberOfAllClusters))

		for _, runtimeData := range runtimesPage.Data {
			verifyKEBAllClustersCountMetricValue(expectedMetricValue, g, runtimeData)
		}
	})

	t.Run("with loaded cache followed by deprovisioning completely(with empty runtimes in KEB response)", func(t *testing.T) {
		// Reset the cluster count necessary for clean slate of next tests
		kebFetchedClusters.Reset()

		subAccID := uuid.New().String()
		cache := gocache.New(gocache.NoExpiration, gocache.NoExpiration)
		queue := workqueue.NewTypedDelayingQueue[string]()
		oldShootName := fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5))

		p := Process{
			Queue:  queue,
			Cache:  cache,
			Logger: logger.NewLogger(zapcore.InfoLevel),
		}
		oldRecord := NewRecord(subAccID, oldShootName, "foo")

		err := p.Cache.Add(subAccID, oldRecord, gocache.NoExpiration)
		g.Expect(err).Should(gomega.BeNil())

		runtimesPageWithNoRuntimes := new(kebruntime.RuntimesPage)
		expectedEmptyQueue := workqueue.NewTypedDelayingQueue[string]()
		expectedEmptyCache := gocache.New(gocache.NoExpiration, gocache.NoExpiration)

		runtimesPageWithNoRuntimes.Data = []kebruntime.RuntimeDTO{}

		p.populateCacheAndQueue(runtimesPageWithNoRuntimes)
		g.Expect(*p.Cache).To(gomega.Equal(*expectedEmptyCache))
		g.Expect(areQueuesEqual(p.Queue, expectedEmptyQueue)).To(gomega.BeTrue())

		// Ensure metric exists
		metricName := fecthedClustersMetricName
		numberOfAllClusters := 0
		expectedMetricValue := 0

		g.Eventually(testutil.CollectAndCount(kebFetchedClusters, metricName)).Should(gomega.Equal(numberOfAllClusters))

		for _, runtimeData := range runtimesPageWithNoRuntimes.Data {
			verifyKEBAllClustersCountMetricValue(expectedMetricValue, g, runtimeData)
		}
	})

	t.Run("with loaded cache, then shoot is deprovisioned and provisioned again", func(t *testing.T) {
		// Reset the cluster count necessary for clean slate of next tests
		kebFetchedClusters.Reset()

		subAccID := uuid.New().String()
		cache := gocache.New(gocache.NoExpiration, gocache.NoExpiration)
		queue := workqueue.NewTypedDelayingQueue[string]()
		oldShootName := fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5))
		newShootName := fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5))

		p := Process{
			Queue:  queue,
			Cache:  cache,
			Logger: logger.NewLogger(zapcore.InfoLevel),
		}
		oldRecord := NewRecord(subAccID, oldShootName, "foo")

		err := p.Cache.Add(subAccID, oldRecord, gocache.NoExpiration)
		g.Expect(err).Should(gomega.BeNil())

		runtimesPage := new(kebruntime.RuntimesPage)
		expectedQueue := workqueue.NewTypedDelayingQueue[string]()
		expectedCache := gocache.New(gocache.NoExpiration, gocache.NoExpiration)

		rntme := kmctesting.NewRuntimesDTO(subAccID, oldShootName, kmctesting.WithProvisionedAndDeprovisionedStatus(kebruntime.StateDeprovisioned))
		runtimesPage.Data = append(runtimesPage.Data, rntme)

		// expected cache changes after deprovisioning
		p.populateCacheAndQueue(runtimesPage)
		g.Expect(*p.Cache).To(gomega.Equal(*expectedCache))
		g.Expect(areQueuesEqual(p.Queue, expectedQueue)).To(gomega.BeTrue())

		// provision a new SKR again with a new name
		skrRuntimesPageWithProvisioning := new(kebruntime.RuntimesPage)
		skrRuntimesPageWithProvisioning.Data = []kebruntime.RuntimeDTO{
			kmctesting.NewRuntimesDTO(subAccID, newShootName, kmctesting.WithProvisioningSucceededStatus(kebruntime.StateSucceeded)),
		}

		// expected cache changes after provisioning
		newRecord := NewRecord(subAccID, newShootName, "")
		err = expectedCache.Add(subAccID, newRecord, gocache.NoExpiration)
		g.Expect(err).Should(gomega.BeNil())

		runtimesPage.Data = []kebruntime.RuntimeDTO{rntme}

		p.populateCacheAndQueue(skrRuntimesPageWithProvisioning)
		g.Expect(*p.Cache).To(gomega.Equal(*expectedCache))
		gotSubAccID, _ := p.Queue.Get()
		g.Expect(gotSubAccID).To(gomega.Equal(subAccID))

		// Ensure metric exists
		metricName := fecthedClustersMetricName
		// expecting number of all clusters to be 1, as deprovisioned shoot is removed
		// only counting the new shoot
		numberOfAllClusters := 1
		g.Eventually(testutil.CollectAndCount(kebFetchedClusters, metricName)).Should(gomega.Equal(numberOfAllClusters))
		// old shoot should not be present in the metric
		for _, runtimeData := range runtimesPage.Data {
			expectedMetricValue := 0

			switch shootName := runtimeData.ShootName; shootName {
			case oldShootName:
				expectedMetricValue = 0
			case newShootName:
				expectedMetricValue = 1
			}

			verifyKEBAllClustersCountMetricValue(expectedMetricValue, g, runtimeData)
		}
	})
}

// TestPrometheusMetricsRemovedForDeletedSubAccounts tests that the prometheus metrics
// are deleted by `populateCacheAndQueue` method. It will test the following cases:
// case 1: Cache entry exists for a shoot, but it is not returned by KEB anymore.
// case 2: Shoot with de-provisioned status returned by KEB.
// case 3: Shoot name of existing subAccount changed and cache entry exists with old shoot name.
func TestPrometheusMetricsRemovedForDeletedSubAccounts(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	// test cases. These cases are not safe to be run in parallel.
	testCases := []struct {
		name                       string
		givenShoot1                kmccache.Record
		givenShoot2                kmccache.Record
		givenShoot2NewName         string
		givenIsShoot2ReturnedByKEB bool
	}{
		{
			name: "should have removed metrics when cache entry exists for a shoot, but it is not returned by KEB anymore",
			givenShoot1: kmccache.Record{
				SubAccountID:    uuid.New().String(),
				InstanceID:      uuid.New().String(),
				RuntimeID:       uuid.New().String(),
				GlobalAccountID: uuid.New().String(),
				ShootName:       fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5)),
				ProviderType:    config.Azure,
			},
			givenShoot2: kmccache.Record{
				SubAccountID:    uuid.New().String(),
				InstanceID:      uuid.New().String(),
				RuntimeID:       uuid.New().String(),
				GlobalAccountID: uuid.New().String(),
				ShootName:       fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5)),
				ProviderType:    config.Azure,
			},
			givenIsShoot2ReturnedByKEB: false,
		},
		{
			name: "should have removed metrics when cache entry exists for a shoot, but KEB returns shoot with de-provisioned status",
			givenShoot1: kmccache.Record{
				SubAccountID:    uuid.New().String(),
				InstanceID:      uuid.New().String(),
				RuntimeID:       uuid.New().String(),
				GlobalAccountID: uuid.New().String(),
				ShootName:       fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5)),
				ProviderType:    config.Azure,
			},
			givenShoot2: kmccache.Record{
				SubAccountID:    uuid.New().String(),
				InstanceID:      uuid.New().String(),
				RuntimeID:       uuid.New().String(),
				GlobalAccountID: uuid.New().String(),
				ShootName:       fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5)),
				ProviderType:    config.Azure,
			},
			givenIsShoot2ReturnedByKEB: true,
		},
		{
			name: "should have removed metrics when cache entry exists for a shoot, but KEB returns shoot with different shoot name",
			givenShoot1: kmccache.Record{
				SubAccountID:    uuid.New().String(),
				InstanceID:      uuid.New().String(),
				RuntimeID:       uuid.New().String(),
				GlobalAccountID: uuid.New().String(),
				ShootName:       fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5)),
				ProviderType:    config.Azure,
			},
			givenShoot2: kmccache.Record{
				SubAccountID:    uuid.New().String(),
				InstanceID:      uuid.New().String(),
				RuntimeID:       uuid.New().String(),
				GlobalAccountID: uuid.New().String(),
				ShootName:       fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5)),
				ProviderType:    config.Azure,
			},
			givenIsShoot2ReturnedByKEB: true,
			givenShoot2NewName:         fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5)),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			// reset metrics.
			subAccountProcessed.Reset()
			subAccountProcessedTimeStamp.Reset()
			collector.TotalScans.Reset()
			collector.TotalScansConverted.Reset()

			// add metrics for both shoots.
			recordSubAccountProcessed(false, tc.givenShoot1)
			recordSubAccountProcessed(false, tc.givenShoot2)

			recordSubAccountProcessedTimeStamp(tc.givenShoot1)
			recordSubAccountProcessedTimeStamp(tc.givenShoot2)

			resourceName := "node"
			backendName := "edp"
			shoot1RuntimeInfo := runtime2.Info{
				InstanceID:      tc.givenShoot1.InstanceID,
				RuntimeID:       tc.givenShoot1.RuntimeID,
				SubAccountID:    tc.givenShoot1.SubAccountID,
				GlobalAccountID: tc.givenShoot1.GlobalAccountID,
				ShootName:       tc.givenShoot1.ShootName,
			}
			shoot2RuntimeInfo := runtime2.Info{
				InstanceID:      tc.givenShoot2.InstanceID,
				RuntimeID:       tc.givenShoot2.RuntimeID,
				SubAccountID:    tc.givenShoot2.SubAccountID,
				GlobalAccountID: tc.givenShoot2.GlobalAccountID,
				ShootName:       tc.givenShoot2.ShootName,
			}

			collector.RecordScan(false, resourceName, shoot1RuntimeInfo)
			collector.RecordScan(false, resourceName, shoot2RuntimeInfo)

			collector.RecordScanConversion(false, resourceName, backendName, shoot1RuntimeInfo)
			collector.RecordScanConversion(false, resourceName, backendName, shoot2RuntimeInfo)

			// setup cache
			cache := gocache.New(gocache.NoExpiration, gocache.NoExpiration)

			// add both shoots to cache
			err := cache.Add(tc.givenShoot1.SubAccountID, tc.givenShoot1, gocache.NoExpiration)
			g.Expect(err).Should(gomega.BeNil())

			err = cache.Add(tc.givenShoot2.SubAccountID, tc.givenShoot2, gocache.NoExpiration)
			g.Expect(err).Should(gomega.BeNil())

			// init queue.
			queue := workqueue.NewTypedDelayingQueue[string]()
			expectedCache := gocache.New(gocache.NoExpiration, gocache.NoExpiration)
			err = expectedCache.Add(tc.givenShoot1.SubAccountID, tc.givenShoot1, gocache.NoExpiration)
			g.Expect(err).Should(gomega.BeNil())

			// mock KEB response.
			runtimesPage := new(kebruntime.RuntimesPage)
			runtime := kmctesting.NewRuntimesDTO(tc.givenShoot1.SubAccountID,
				tc.givenShoot1.ShootName, kmctesting.WithProvisioningSucceededStatus(kebruntime.StateSucceeded))
			runtimesPage.Data = append(runtimesPage.Data, runtime)

			if tc.givenIsShoot2ReturnedByKEB {
				runtime = kmctesting.NewRuntimesDTO(tc.givenShoot2.SubAccountID,
					tc.givenShoot2.ShootName, kmctesting.WithProvisionedAndDeprovisionedStatus(kebruntime.StateDeprovisioned))
				if tc.givenShoot2NewName != "" {
					runtime = kmctesting.NewRuntimesDTO(tc.givenShoot2.SubAccountID,
						tc.givenShoot2NewName, kmctesting.WithProvisioningSucceededStatus(kebruntime.StateSucceeded))
				}

				runtimesPage.Data = append(runtimesPage.Data, runtime)
			}

			p := Process{
				Queue:  queue,
				Cache:  cache,
				Logger: logger.NewLogger(zapcore.InfoLevel),
			}

			// when
			p.populateCacheAndQueue(runtimesPage)

			// then
			// check if metrics still exists or not for shoot1 (existing shoot)
			// metric: subAccountProcessed
			gotMetrics, err := subAccountProcessed.GetMetricWithLabelValues(
				strconv.FormatBool(false),
				tc.givenShoot1.ShootName,
				tc.givenShoot1.InstanceID,
				tc.givenShoot1.RuntimeID,
				tc.givenShoot1.SubAccountID,
				tc.givenShoot1.GlobalAccountID,
			)
			g.Expect(err).Should(gomega.BeNil())
			g.Expect(testutil.ToFloat64(gotMetrics)).Should(gomega.Equal(float64(1)))
			// metric: subAccountProcessedTimeStamp
			gotMetrics, err = subAccountProcessedTimeStamp.GetMetricWithLabelValues(
				tc.givenShoot1.ShootName,
				tc.givenShoot1.InstanceID,
				tc.givenShoot1.RuntimeID,
				tc.givenShoot1.SubAccountID,
				tc.givenShoot1.GlobalAccountID,
			)
			g.Expect(err).Should(gomega.BeNil())
			g.Expect(testutil.ToFloat64(gotMetrics)).ShouldNot(gomega.Equal(float64(0)))
			// metric: TotalScans
			gotMetrics, err = collector.TotalScans.GetMetricWithLabelValues(
				strconv.FormatBool(false),
				resourceName,
				tc.givenShoot1.ShootName,
				tc.givenShoot1.InstanceID,
				tc.givenShoot1.RuntimeID,
				tc.givenShoot1.SubAccountID,
				tc.givenShoot1.GlobalAccountID,
			)
			g.Expect(err).Should(gomega.BeNil())
			g.Expect(testutil.ToFloat64(gotMetrics)).Should(gomega.Equal(float64(1)))
			// metric: TotalScansConverted
			gotMetrics, err = collector.TotalScansConverted.GetMetricWithLabelValues(
				strconv.FormatBool(false),
				resourceName,
				backendName,
				tc.givenShoot1.ShootName,
				tc.givenShoot1.InstanceID,
				tc.givenShoot1.RuntimeID,
				tc.givenShoot1.SubAccountID,
				tc.givenShoot1.GlobalAccountID,
			)
			g.Expect(err).Should(gomega.BeNil())
			g.Expect(testutil.ToFloat64(gotMetrics)).Should(gomega.Equal(float64(1)))

			// check if metrics were deleted or not for shoot2 which is not returned by KEB anymore OR returned with de-provisioned status OR returned with different shoot name
			// metric: subAccountProcessed
			gotMetrics, err = subAccountProcessed.GetMetricWithLabelValues(
				strconv.FormatBool(false),
				tc.givenShoot2.ShootName,
				tc.givenShoot2.InstanceID,
				tc.givenShoot2.RuntimeID,
				tc.givenShoot2.SubAccountID,
				tc.givenShoot2.GlobalAccountID,
			)
			g.Expect(err).Should(gomega.BeNil())
			g.Expect(testutil.ToFloat64(gotMetrics)).Should(gomega.Equal(float64(0)))
			// metric: subAccountProcessedTimeStamp
			gotMetrics, err = subAccountProcessedTimeStamp.GetMetricWithLabelValues(
				tc.givenShoot2.ShootName,
				tc.givenShoot2.InstanceID,
				tc.givenShoot2.RuntimeID,
				tc.givenShoot2.SubAccountID,
				tc.givenShoot2.GlobalAccountID,
			)
			g.Expect(err).Should(gomega.BeNil())
			g.Expect(testutil.ToFloat64(gotMetrics)).Should(gomega.Equal(float64(0)))
			// metric: TotalScans
			gotMetrics, err = collector.TotalScans.GetMetricWithLabelValues(
				strconv.FormatBool(false),
				resourceName,
				tc.givenShoot2.ShootName,
				tc.givenShoot2.InstanceID,
				tc.givenShoot2.RuntimeID,
				tc.givenShoot2.SubAccountID,
				tc.givenShoot2.GlobalAccountID,
			)
			g.Expect(err).Should(gomega.BeNil())
			g.Expect(testutil.ToFloat64(gotMetrics)).Should(gomega.Equal(float64(0)))
			// metric: TotalScansConverted
			gotMetrics, err = collector.TotalScansConverted.GetMetricWithLabelValues(
				strconv.FormatBool(false),
				resourceName,
				backendName,
				tc.givenShoot2.ShootName,
				tc.givenShoot2.InstanceID,
				tc.givenShoot2.RuntimeID,
				tc.givenShoot2.SubAccountID,
				tc.givenShoot2.GlobalAccountID,
			)
			g.Expect(err).Should(gomega.BeNil())
			g.Expect(testutil.ToFloat64(gotMetrics)).Should(gomega.Equal(float64(0)))
		})
	}
}

// TestPrometheusMetricsProcessSubAccountID tests the prometheus metrics maintained by `processSubAccountID` method.
func TestPrometheusMetricsProcessSubAccountID(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	// given (common for all test cases).
	logger := logger.NewLogger(zapcore.DebugLevel)

	const givenMethodRecalls = 3

	givenKubeConfig := generateFakeKubeConfig()
	subAccountID := uuid.New().String()

	// test cases. These cases are not safe to be run in parallel.
	testCases := []struct {
		name              string
		givenShoot        kmccache.Record
		EDPCollector      collector.CollectorSender
		KubeConfig        string
		expectedToSucceed bool
	}{
		{
			name: "should have correct metrics when it successfully processes subAccount",
			givenShoot: kmccache.Record{
				SubAccountID:    subAccountID,
				InstanceID:      uuid.New().String(),
				RuntimeID:       uuid.New().String(),
				GlobalAccountID: uuid.New().String(),
				ShootName:       fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5)),
			},
			KubeConfig:        givenKubeConfig,
			EDPCollector:      stubs.NewCollector(nil, nil),
			expectedToSucceed: true,
		},
		{
			name: "should have correct metrics when it fails to process subAccount",
			givenShoot: kmccache.Record{
				SubAccountID:    subAccountID,
				InstanceID:      uuid.New().String(),
				RuntimeID:       uuid.New().String(),
				GlobalAccountID: uuid.New().String(),
				ShootName:       fmt.Sprintf("shoot-%s", kmctesting.GenerateRandomAlphaString(5)),
			},
			KubeConfig:        givenKubeConfig,
			EDPCollector:      stubs.NewCollector(nil, fmt.Errorf("fake error")),
			expectedToSucceed: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testStartTimeUnix := time.Now().Unix()

			subAccountProcessed.Reset()
			subAccountProcessedTimeStamp.Reset()

			// populate cache.
			cache := gocache.New(gocache.NoExpiration, gocache.NoExpiration)
			err := cache.Add(tc.givenShoot.SubAccountID, tc.givenShoot, gocache.NoExpiration)
			g.Expect(err).Should(gomega.BeNil())

			secretKCPStored := kmctesting.NewKCPStoredSecret(tc.givenShoot.RuntimeID, tc.KubeConfig)
			secretCacheClient := fake.NewSimpleClientset(secretKCPStored)

			// initiate process instance.
			givenProcess := &Process{
				EDPCollector:      tc.EDPCollector,
				Queue:             workqueue.NewTypedDelayingQueue[string](),
				SecretCacheClient: secretCacheClient.CoreV1(),
				Cache:             cache,
				ScrapeInterval:    3 * time.Second,
				Logger:            logger,
			}

			// when
			// calling the method multiple times to generate testable metrics.
			for i := range givenMethodRecalls {
				givenProcess.processSubAccountID(tc.givenShoot.SubAccountID, i)
			}

			// then
			// check prometheus metrics.
			// metric: subAccountProcessed
			gotMetrics, err := subAccountProcessed.GetMetricWithLabelValues(
				strconv.FormatBool(tc.expectedToSucceed),
				tc.givenShoot.ShootName,
				tc.givenShoot.InstanceID,
				tc.givenShoot.RuntimeID,
				tc.givenShoot.SubAccountID,
				tc.givenShoot.GlobalAccountID,
			)
			g.Expect(err).Should(gomega.BeNil())
			// the metric will be incremented even in case of failure, so that is why
			// it should be equal to the number of time the `processSubAccountID` is called.
			g.Expect(testutil.ToFloat64(gotMetrics)).Should(gomega.Equal(float64(givenMethodRecalls)))

			// metric: subAccountProcessedTimeStamp
			gotMetrics, err = subAccountProcessedTimeStamp.GetMetricWithLabelValues(
				tc.givenShoot.ShootName,
				tc.givenShoot.InstanceID,
				tc.givenShoot.RuntimeID,
				tc.givenShoot.SubAccountID,
				tc.givenShoot.GlobalAccountID,
			)
			g.Expect(err).Should(gomega.BeNil())
			// check if the last published time has correct value.
			// the timestamp will only be updated when the subAccount is successfully processed.
			utcTime := testutil.ToFloat64(gotMetrics)
			isPublishedAfterTestStartTime := int64(utcTime) >= testStartTimeUnix
			g.Expect(isPublishedAfterTestStartTime).Should(
				gomega.Equal(tc.expectedToSucceed),
				"the last published time should be updated only when a new event is published to EDP.")
		})
	}
}

func TestExecute(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	subAccID := uuid.New().String()
	runtimeID := uuid.New().String()
	expectedKubeconfig := generateFakeKubeConfig()
	log := logger.NewLogger(zapcore.DebugLevel)

	secretKCPStored := kmctesting.NewKCPStoredSecret(runtimeID, expectedKubeconfig)
	secretCacheClient := fake.NewSimpleClientset(secretKCPStored)

	expectedScanMap := NewScanMap()
	collector := stubs.NewCollector(expectedScanMap, nil)

	// Populate cache
	cache := gocache.New(gocache.NoExpiration, gocache.NoExpiration)
	newRecord := kmccache.Record{
		SubAccountID: subAccID,
		RuntimeID:    runtimeID,
		ScanMap:      nil,
	}
	err := cache.Add(subAccID, newRecord, gocache.NoExpiration)
	g.Expect(err).Should(gomega.BeNil())

	// Populate queue
	queue := workqueue.NewTypedDelayingQueue[string]()
	queue.Add(subAccID)

	newProcess := &Process{
		EDPCollector:      collector,
		Queue:             queue,
		SecretCacheClient: secretCacheClient.CoreV1(),
		Cache:             cache,
		ScrapeInterval:    3 * time.Second,
		Logger:            log,
	}

	go func() {
		newProcess.execute(1)
	}()

	// Test cache state
	g.Eventually(func() error {
		itemFromCache, found := newProcess.Cache.Get(subAccID)
		if !found {
			return fmt.Errorf("subAccID not found in cache")
		}

		record, ok := itemFromCache.(kmccache.Record)
		if !ok {
			return fmt.Errorf("failed to cast item from cache to type kmccache.Record")
		}

		if !reflect.DeepEqual(record.ScanMap, expectedScanMap) {
			return fmt.Errorf("record scan map mismatch, got: %v, expected: %v", record.ScanMap, expectedScanMap)
		}

		return nil
	}, bigTimeout).Should(gomega.BeNil())

	// Clean it from the cache once SKR is deprovisioned
	newProcess.Cache.Delete(subAccID)

	go func() {
		newProcess.execute(1)
	}()

	time.Sleep(timeout)
	// the queue should be empty.
	g.Eventually(newProcess.Queue.Len()).Should(gomega.Equal(0))
}

func NewRecord(subAccId, shootName, kubeconfig string) kmccache.Record {
	return kmccache.Record{
		SubAccountID: subAccId,
		ShootName:    shootName,
		ScanMap:      nil,
	}
}

func areQueuesEqual(src, dest workqueue.TypedDelayingInterface[string]) bool {
	if src.Len() != dest.Len() {
		return false
	}

	for src.Len() > 0 {
		srcItem, _ := src.Get()
		destItem, _ := dest.Get()

		if srcItem != destItem {
			return false
		}
	}

	return true
}

func AddSuccessfulIDsToCacheQueueAndRuntimes(runtimesPage *kebruntime.RuntimesPage, successfulIDs []string, expectedCache *gocache.Cache, expectedQueue workqueue.TypedDelayingInterface[string]) (*kebruntime.RuntimesPage, *gocache.Cache, workqueue.TypedDelayingInterface[string], error) {
	for _, successfulID := range successfulIDs {
		shootID := kmctesting.GenerateRandomAlphaString(5)
		shootName := fmt.Sprintf("shoot-%s", shootID)
		runtime := kmctesting.NewRuntimesDTO(successfulID, shootName, kmctesting.WithProvisioningSucceededStatus(kebruntime.StateSucceeded))
		runtimesPage.Data = append(runtimesPage.Data, runtime)

		err := expectedCache.Add(successfulID, kmccache.Record{
			SubAccountID: successfulID,
			ShootName:    shootName,
		}, gocache.NoExpiration)
		if err != nil {
			return nil, nil, nil, err
		}

		expectedQueue.Add(successfulID)
	}

	return runtimesPage, expectedCache, expectedQueue, nil
}

func NewScanMap() collector.ScanMap {
	scans := make(collector.ScanMap)

	scans["node"] = stubs.NewScan([]string{"node1", "node2"})
	scans["redis"] = stubs.NewScan([]string{"redis1", "redis2"})

	return scans
}

// Helper function to check the value of the `kmc_process_fetched_clusters` metric using `ToFloat64`.
func verifyKEBAllClustersCountMetricValue(expectedValue int, g *gomega.WithT, runtimeData kebruntime.RuntimeDTO) bool {
	return g.Eventually(func() int {
		trackable := isRuntimeTrackable(runtimeData)

		counter, err := kebFetchedClusters.GetMetricWithLabelValues(
			strconv.FormatBool(trackable),
			runtimeData.ShootName,
			runtimeData.InstanceID,
			runtimeData.RuntimeID,
			runtimeData.SubAccountID,
			runtimeData.GlobalAccountID)

		g.Expect(err).Should(gomega.BeNil())
		// check the value of the metric
		return int(testutil.ToFloat64(counter))
	}).Should(gomega.Equal(expectedValue))
}

// generateFakeKubeConfig generates a fake kubeconfig content as a string.
func generateFakeKubeConfig() string {
	return `
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: ` + base64.StdEncoding.EncodeToString([]byte("fake-ca-data")) + `
    server: https://fake-server:6443
  name: fake-cluster
contexts:
- context:
    cluster: fake-cluster
    user: fake-user
  name: fake-context
current-context: fake-context
kind: Config
preferences: {}
users:
- name: fake-user
  user:
    client-certificate-data: ` + base64.StdEncoding.EncodeToString([]byte("fake-client-cert-data")) + `
    client-key-data: ` + base64.StdEncoding.EncodeToString([]byte("fake-client-key-data")) + `
`
}
