package process

import (
	"context"
	"fmt"

	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
	"k8s.io/client-go/tools/clientcmd"

	kmccache "github.com/kyma-project/kyma-metrics-collector/pkg/kubeconfigprovider"
	log "github.com/kyma-project/kyma-metrics-collector/pkg/logger"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
)

func (p *Process) processSubAccountID(subAccountID string, identifier int) bool {
	p.queueProcessingLogger(nil, subAccountID, identifier).
		Debug("fetched subAccountID from queue")

	// Get the kubeconfigprovider item for the subAccountID
	cacheItem, exists := p.Cache.Get(subAccountID)
	if !exists {
		p.queueProcessingLogger(nil, subAccountID, identifier).With(log.KeyRequeue, log.ValueFalse).
			Info("subAccountID is not found in kubeconfigprovider which means it is not trackable anymore")

		recordSubAccountProcessed(false, kmccache.Record{SubAccountID: subAccountID})

		return false
	}

	// Cast the kubeconfigprovider item to a Record object
	var record kmccache.Record

	var ok bool

	record, ok = cacheItem.(kmccache.Record)
	if !ok {
		p.handleError(nil, subAccountID, identifier, fmt.Errorf("bad item from kubeconfigprovider, could not cast it to a record obj"))

		return false
	}

	p.queueProcessingLogger(&record, subAccountID, identifier).
		Debugf("record found from kubeconfigprovider: %+v", record)

	// Get kubeConfig from kubeconfigprovider
	kubeConfig, err := p.KubeconfigProvider.Get(record.RuntimeID)
	if err != nil {
		p.handleError(&record, subAccountID, identifier, fmt.Errorf("failed to load kubeconfig from kubeconfigprovider: %w", err))

		return false
	}

	// Create REST client config from kubeConfig
	restClientConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeConfig)
	if err != nil {
		p.handleError(&record, subAccountID, identifier, fmt.Errorf("failed to create REST config from kubeconfig: %w", err))

		return false
	}

	// Collect and send measurements to EDP backend
	ctx := context.Background()
	runtimeInfo := runtime.Info{
		InstanceID:      record.InstanceID,
		RuntimeID:       record.RuntimeID,
		SubAccountID:    record.SubAccountID,
		GlobalAccountID: record.GlobalAccountID,
		ShootName:       record.ShootName,
		ProviderType:    record.ProviderType,
		Kubeconfig:      *restClientConfig,
	}

	newScans, err := p.EDPCollector.CollectAndSend(ctx, &runtimeInfo, record.ScanMap)
	if err != nil {
		p.handleError(&record, subAccountID, identifier, fmt.Errorf("failed to collect and send measurements to EDP backend: %w", err))

		return false
	}

	record.ScanMap = newScans
	p.queueProcessingLogger(&record, subAccountID, identifier).
		Info("successfully collected and sent measurements to EDP backend")

	// Record metrics
	recordSubAccountProcessed(true, record)
	recordSubAccountProcessedTimeStamp(record)

	// Update kubeconfigprovider
	p.Cache.Set(record.SubAccountID, record, cache.NoExpiration)
	p.queueProcessingLogger(&record, subAccountID, identifier).
		Debug("updated kubeconfigprovider with new record")

	return true
}

func (p *Process) handleError(record *kmccache.Record, subAccountID string, identifier int, err error) {
	p.queueProcessingLogger(record, subAccountID, identifier).
		Errorf(err.Error())

	p.Queue.AddAfter(subAccountID, p.ScrapeInterval)

	p.queueProcessingLogger(record, subAccountID, identifier).With(log.KeyRequeue, log.ValueTrue).
		Debugf("successfully requeued subAccountID after %v", p.ScrapeInterval)

	recordSubAccountProcessed(false, *record)
}

func (p *Process) queueProcessingLogger(record *kmccache.Record, subAccountID string, identifier int) *zap.SugaredLogger {
	logger := p.Logger.With("component", "kmc").With(log.KeyWorkerID, identifier).With(log.KeySubAccountID, subAccountID)
	if record == nil {
		return logger
	}

	return logger.With(log.KeyRuntimeID, record.RuntimeID).With(log.KeyShoot, record.ShootName).With(log.KeyGlobalAccountID, record.GlobalAccountID)
}
