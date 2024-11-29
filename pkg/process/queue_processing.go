package process

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	kebruntime "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"

	kmccache "github.com/kyma-project/kyma-metrics-collector/pkg/cache"
	edpcollector "github.com/kyma-project/kyma-metrics-collector/pkg/collector/edp"
	log "github.com/kyma-project/kyma-metrics-collector/pkg/logger"
	"github.com/kyma-project/kyma-metrics-collector/pkg/resource/node"
	"github.com/kyma-project/kyma-metrics-collector/pkg/resource/pvc"
	"github.com/kyma-project/kyma-metrics-collector/pkg/resource/redis"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
	skrredis "github.com/kyma-project/kyma-metrics-collector/pkg/skr/redis"
)

const (
	testPublicCloudSpecsPath = "../testing/fixtures/public_cloud_specs.json"
)

func (p *Process) generateRecordWithNewMetrics(identifier int, subAccountID string) (kmccache.Record, error) {
	ctx := context.Background()

	var ok bool

	obj, isFound := p.Cache.Get(subAccountID)
	if !isFound {
		err := errSubAccountIDNotTrackable

		return kmccache.Record{
			SubAccountID: subAccountID,
		}, err
	}

	var record kmccache.Record

	if record, ok = obj.(kmccache.Record); !ok {
		return kmccache.Record{SubAccountID: subAccountID}, errBadItemFromCache
	}

	p.namedLogger().With(log.KeyWorkerID, identifier).Debugf("record found from cache: %+v", record)

	runtimeID := record.RuntimeID

	kubeconfig, err := kmccache.GetKubeConfigFromCache(p.Logger, p.SecretCacheClient, runtimeID)
	if err != nil {
		return record, fmt.Errorf("%w: %w", ErrLoadingFailed, err)
	}

	record.KubeConfig = kubeconfig

	// Get nodes dynamic client
	nodesClient, err := p.NodeConfig.NewClient(record)
	if err != nil {
		return record, err
	}

	// Get nodes
	var nodes *corev1.NodeList

	nodes, err = nodesClient.List(ctx)
	if err != nil {
		return record, err
	}

	if len(nodes.Items) == 0 {
		err = fmt.Errorf("no nodes to process")
		return record, err
	}

	// Get PVCs
	pvcClient, err := p.PVCConfig.NewClient(record)
	if err != nil {
		return record, err
	}

	var pvcList *corev1.PersistentVolumeClaimList

	pvcList, err = pvcClient.List(ctx)
	if err != nil {
		return record, err
	}

	// Get Svcs
	var svcList *corev1.ServiceList

	svcClient, err := p.SvcConfig.NewClient(record)
	if err != nil {
		return record, err
	}

	svcList, err = svcClient.List(ctx)
	if err != nil {
		return record, err
	}

	// Get Redis resources
	var redisList *skrredis.RedisList

	redisClient, err := p.RedisConfig.NewClient(record)
	if err != nil {
		return record, err
	}

	redisList, err = redisClient.List(ctx)
	if err != nil {
		return record, err
	}

	// Create input
	input := Input{
		provider:  record.ProviderType,
		nodeList:  nodes,
		pvcList:   pvcList,
		svcList:   svcList,
		redisList: redisList,
	}

	metric, err := input.Parse(p.PublicCloudSpecs)
	if err != nil {
		return record, err
	}

	metric.RuntimeId = record.RuntimeID
	metric.SubAccountId = record.SubAccountID
	metric.ShootName = record.ShootName
	record.Metric = metric

	// Running new collector in parallel to test OTel instrumentation
	restClientConfig, _ := clientcmd.RESTConfigFromKubeConfig([]byte(record.KubeConfig))

	collector := edpcollector.NewCollector(
		node.NewScanner(p.PublicCloudSpecs),
		redis.NewScanner(p.PublicCloudSpecs),
		pvc.NewScanner(),
	)
	collector.CollectAndSend(
		ctx,
		&runtime.Info{
			Kubeconfig:   *restClientConfig,
			ProviderType: runtime.ProviderType(record.ProviderType),
			ShootID:      record.ShootName,
		},
		nil,
	)

	return record, nil
}

// getOldRecordIfMetricExists gets old record from cache if old metric exists.
func (p *Process) getOldRecordIfMetricExists(subAccountID string) (*kmccache.Record, error) {
	oldRecordObj, found := p.Cache.Get(subAccountID)
	if !found {
		notFoundErr := fmt.Errorf("subAccountID: %s not found", subAccountID)
		p.Logger.Error(notFoundErr)

		return nil, notFoundErr
	}

	if oldRecord, ok := oldRecordObj.(kmccache.Record); ok {
		if oldRecord.Metric != nil {
			return &oldRecord, nil
		}
	}

	notFoundErr := fmt.Errorf("old metrics for subAccountID: %s not found", subAccountID)
	p.Logger.With(log.KeySubAccountID, subAccountID).Error("old metrics for subAccount not found")

	return nil, notFoundErr
}

func (p *Process) processSubAccountID(subAccountID string, identifier int) {
	var payload []byte

	if strings.TrimSpace(subAccountID) == "" {
		p.namedLogger().With(log.KeyWorkerID, identifier).Warn("cannot work with empty subAccountID")

		// Nothing to do further
		return
	}

	p.namedLogger().With(log.KeySubAccountID, subAccountID).With(log.KeyWorkerID, identifier).
		Debug("fetched subAccountID from queue")

	record, isOldMetricValid, err := p.getRecordWithOldOrNewMetric(identifier, subAccountID)
	if err != nil {
		p.namedLoggerWithRecord(record).
			With(log.KeyResult, log.ValueFail).
			With(log.KeyError, err.Error()).
			With(log.KeyWorkerID, identifier).
			With(log.KeySubAccountID, subAccountID).
			Error("no metric found/generated for subaccount")
		// SubAccountID is not trackable anymore as there is no runtime
		if errors.Is(err, errSubAccountIDNotTrackable) {
			p.namedLoggerWithRecord(record).
				With(log.KeyRequeue, log.ValueFalse).
				With(log.KeyWorkerID, identifier).
				With(log.KeySubAccountID, subAccountID).
				Info("subAccountID NOT requeued")

			recordSubAccountProcessed(false, *record)

			return
		}

		p.Queue.AddAfter(subAccountID, p.ScrapeInterval)
		p.namedLoggerWithRecord(record).
			With(log.KeyRequeue, log.ValueTrue).
			With(log.KeySubAccountID, subAccountID).
			With(log.KeyWorkerID, identifier).
			Debugf("successfully requeued subAccountID after %v", p.ScrapeInterval)

		// record metric.
		recordSubAccountProcessed(false, *record)

		// Nothing to do further
		return
	}

	// Convert metric to JSON
	payload, err = json.Marshal(*record.Metric)
	if err != nil {
		p.namedLoggerWithRecord(record).
			With(log.KeyResult, log.ValueFail).
			With(log.KeyError, err.Error()).
			With(log.KeyWorkerID, identifier).
			With(log.KeySubAccountID, subAccountID).
			Error("json.Marshal metric for subAccountID")

		p.Queue.AddAfter(subAccountID, p.ScrapeInterval)
		p.namedLoggerWithRecord(record).
			With(log.KeyResult, log.ValueSuccess).
			With(log.KeyRequeue, log.ValueTrue).
			With(log.KeyWorkerID, identifier).
			With(log.KeySubAccountID, subAccountID).
			Debugf("requeued subAccountID after %v", p.ScrapeInterval)

		// record metric.
		recordSubAccountProcessed(false, *record)

		// Nothing to do further
		return
	}

	// Send metrics to EDP
	// Note: EDP refers SubAccountID as tenant
	p.namedLoggerWithRecord(record).
		With(log.KeyWorkerID, identifier).
		Debugf("sending EventStreamToEDP: payload: %s", string(payload))

	err = p.sendEventStreamToEDP(subAccountID, payload)
	if err != nil {
		p.namedLoggerWithRecord(record).
			With(log.KeyResult, log.ValueFail).
			With(log.KeyError, err.Error()).
			With(log.KeyWorkerID, identifier).
			Errorf("send metric to EDP for event-stream: %s", string(payload))

		p.Queue.AddAfter(subAccountID, p.ScrapeInterval)
		p.namedLoggerWithRecord(record).
			With(log.KeyResult, log.ValueSuccess).
			With(log.KeyRequeue, log.ValueTrue).
			With(log.KeyWorkerID, identifier).
			Debugf("requeued subAccountID after %v", p.ScrapeInterval)

		// record metric.
		recordSubAccountProcessed(false, *record)

		// Nothing to do further hence continue
		return
	}

	p.namedLoggerWithRecord(record).
		With(log.KeyResult, log.ValueSuccess).
		With(log.KeyWorkerID, identifier).
		Infof("sent event stream, shoot: %s", record.ShootName)

	// record metrics.
	recordSubAccountProcessed(true, *record)
	recordSubAccountProcessedTimeStamp(isOldMetricValid, *record)

	// update cache.
	if !isOldMetricValid {
		p.Cache.Set(record.SubAccountID, *record, cache.NoExpiration)
		p.namedLoggerWithRecord(record).
			With(log.KeyResult, log.ValueSuccess).
			With(log.KeyWorkerID, identifier).
			Debug("saved metric")
		resetOldMetricsPublishedGauge(*record)
	} else {
		// record metric.
		recordOldMetricsPublishedGauge(*record)
	}

	// Requeue the subAccountID anyway
	p.namedLoggerWithRecord(record).
		With(log.KeyResult, log.ValueSuccess).
		With(log.KeyRequeue, log.ValueTrue).
		With(log.KeyWorkerID, identifier).
		Debugf("requeued subAccountID after %v", p.ScrapeInterval)
	p.Queue.AddAfter(subAccountID, p.ScrapeInterval)
}

// getRecordWithOldOrNewMetric generates new metric or fetches the old metric along with a bool flag which
// indicates whether it is an old metric or not(true, when it is old and false when it is new).
// it always returns a record for metadata.
func (p *Process) getRecordWithOldOrNewMetric(identifier int, subAccountID string) (*kmccache.Record, bool, error) {
	record, err := p.generateRecordWithNewMetrics(identifier, subAccountID)
	if err != nil {
		if errors.Is(err, errSubAccountIDNotTrackable) {
			p.namedLoggerWithRecord(&record).
				With(log.KeyWorkerID, identifier).Info("subAccountID is not trackable anymore, skipping the fetch of old metric")
			return &record, false, err // SubAccountID is not trackable anymore, record returned for metadata
		}

		p.namedLoggerWithRecord(&record).With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).
			Error("generate new metric for subAccount")
		// Get old data
		oldRecord, err := p.getOldRecordIfMetricExists(subAccountID)
		if err != nil {
			// Nothing to do, return the new record for metadata
			return &record, false, errors.Wrapf(err, "failed to get getOldMetric for subaccountID: %s", subAccountID)
		}

		return oldRecord, true, nil
	}

	return &record, false, nil
}

func (p *Process) sendEventStreamToEDP(tenant string, payload []byte) error {
	edpRequest, err := p.EDPClient.NewRequest(tenant)
	if err != nil {
		return errors.Wrapf(err, "failed to create a new request for EDP")
	}

	resp, err := p.EDPClient.Send(edpRequest, payload)
	if err != nil {
		return errors.Wrapf(err, "failed to send event-stream to EDP")
	}

	if !isSuccess(resp.StatusCode) {
		return fmt.Errorf("failed to send event-stream to EDP as it returned HTTP: %d", resp.StatusCode)
	}

	return nil
}

func isSuccess(status int) bool {
	if status >= http.StatusOK && status < http.StatusMultipleChoices {
		return true
	}

	return false
}

func (p *Process) namedLogger() *zap.SugaredLogger {
	return p.Logger.With("component", "kmc")
}

func (p *Process) namedLoggerWithRecord(record *kmccache.Record) *zap.SugaredLogger {
	if record == nil {
		return p.Logger.With("component", "kmc").With(log.KeyRuntimeID, "")
	}

	return p.Logger.With("component", "kmc").With(log.KeyRuntimeID, record.RuntimeID).With(log.KeyShoot, record.ShootName).With(log.KeySubAccountID, record.SubAccountID).With(log.KeyGlobalAccountID, record.GlobalAccountID)
}

func (p *Process) namedLoggerWithRuntime(runtime kebruntime.RuntimeDTO) *zap.SugaredLogger {
	return p.Logger.With("component", "kmc").With(log.KeyRuntimeID, runtime.RuntimeID).With(log.KeyShoot, runtime.ShootName).With(log.KeySubAccountID, runtime.SubAccountID).With(log.KeyGlobalAccountID, runtime.GlobalAccountID)
}
