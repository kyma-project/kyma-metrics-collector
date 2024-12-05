package process

import (
	"context"
	"fmt"
	kmccache "github.com/kyma-project/kyma-metrics-collector/pkg/cache"
	log "github.com/kyma-project/kyma-metrics-collector/pkg/logger"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	testPublicCloudSpecsPath = "../testing/fixtures/public_cloud_specs.json"
)

func (p *Process) processSubAccountID(subAccountID string, identifier int) {
	p.namedLogger().
		With(log.KeyWorkerID, identifier).
		With(log.KeySubAccountID, subAccountID).
		Debug("fetched subAccountID from queue")

	// Get the cache item for the subAccountID
	cacheItem, exists := p.Cache.Get(subAccountID)
	if !exists {
		p.namedLogger().
			With(log.KeyWorkerID, identifier).
			With(log.KeySubAccountID, subAccountID).
			With(log.KeyRequeue, log.ValueFalse).
			Info("subAccountID is not found in cache which means it is not trackable anymore")

		recordSubAccountProcessed(false, kmccache.Record{SubAccountID: subAccountID})

		return
	}

	// Cast the cache item to a Record object
	var record kmccache.Record
	var ok bool
	record, ok = cacheItem.(kmccache.Record)
	if !ok {
		p.namedLogger().
			With(log.KeyWorkerID, identifier).
			With(log.KeySubAccountID, subAccountID).
			Error("bad item from cache, could not cast it to a record obj")

		p.Queue.AddAfter(subAccountID, p.ScrapeInterval)

		p.namedLogger().
			With(log.KeyWorkerID, identifier).
			With(log.KeySubAccountID, subAccountID).
			With(log.KeyRequeue, log.ValueTrue).
			Debugf("successfully requeued subAccountID after %v", p.ScrapeInterval)

		recordSubAccountProcessed(false, record)

		return
	}
	p.namedLogger().
		With(log.KeyWorkerID, identifier).
		With(log.KeySubAccountID, subAccountID).
		Debugf("record found from cache: %+v", record)

	// Get kubeConfig from cache
	kubeConfig, err := kmccache.GetKubeConfigFromCache(p.Logger, p.SecretCacheClient, record.RuntimeID)
	if err != nil {
		p.handleError(&record, subAccountID, identifier, fmt.Sprintf("failed to load kubeconfig from cache: %w", err))
		return
	}
	record.KubeConfig = kubeConfig

	// Create REST client config from kubeConfig
	restClientConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(record.KubeConfig))
	if err != nil {
		p.handleError(&record, subAccountID, identifier, fmt.Sprintf("failed to create REST config from kubeconfig: %w", err))
		return
	}

	// Collect and send measurements to EDP backend
	ctx := context.Background()
	runtimeInfo := runtime.Info{
		Kubeconfig:   *restClientConfig,
		ProviderType: record.ProviderType,
		RuntimeID:    record.RuntimeID,
		SubAccountID: record.RuntimeID,
		ShootName:    record.ShootName,
	}
	newScans, err := p.EDPCollector.CollectAndSend(ctx, &runtimeInfo, *record.Metric)
	if err != nil {
		p.handleError(&record, subAccountID, identifier, fmt.Sprintf("failed to collect and send measurements to EDP backend: %w", err))
		return
	}
	record.Metric = &newScans
	p.namedLoggerWithRecord(&record).
		With(log.KeyWorkerID, identifier).
		With(log.KeySubAccountID, subAccountID).
		Info("successfully collected and sent measurements to EDP backend")

	// Record metrics
	recordSubAccountProcessed(true, record)
	recordSubAccountProcessedTimeStamp(record)

	// Update cache
	p.Cache.Set(record.SubAccountID, record, cache.NoExpiration)
	p.namedLoggerWithRecord(&record).
		With(log.KeyWorkerID, identifier).
		With(log.KeySubAccountID, subAccountID).
		Debug("updated cache with new record")

	// Requeue the subAccountID anyway
	p.Queue.AddAfter(subAccountID, p.ScrapeInterval)
	p.namedLoggerWithRecord(&record).
		With(log.KeyWorkerID, identifier).
		With(log.KeySubAccountID, subAccountID).
		With(log.KeyRequeue, log.ValueTrue).
		Debugf("successfully requeued subAccountID after %v", p.ScrapeInterval)
}

func (p *Process) handleError(record *kmccache.Record, subAccountID string, identifier int, errMsg string) {
	p.namedLoggerWithRecord(record).
		With(log.KeyWorkerID, identifier).
		With(log.KeySubAccountID, subAccountID).
		Errorf(errMsg)

	p.Queue.AddAfter(subAccountID, p.ScrapeInterval)

	p.namedLogger().
		With(log.KeyWorkerID, identifier).
		With(log.KeySubAccountID, subAccountID).
		With(log.KeyRequeue, log.ValueTrue).
		Debugf("successfully requeued subAccountID after %v", p.ScrapeInterval)

	recordSubAccountProcessed(false, *record)
}

func (p *Process) namedLogger() *zap.SugaredLogger {
	return p.Logger.With("component", "kmc")
}

func (p *Process) namedLoggerWithRecord(record *kmccache.Record) *zap.SugaredLogger {
	if record == nil {
		return p.Logger.With("component", "kmc")
	}

	return p.Logger.With("component", "kmc").With(log.KeyRuntimeID, record.RuntimeID).With(log.KeyShoot, record.ShootName).With(log.KeySubAccountID, record.SubAccountID).With(log.KeyGlobalAccountID, record.GlobalAccountID)
}
