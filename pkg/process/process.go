package process

import (
	"fmt"
	"sync"
	"time"

	gocache "github.com/patrickmn/go-cache"
	"go.uber.org/zap"
	"k8s.io/client-go/util/workqueue"

	"github.com/kyma-project/kyma-metrics-collector/pkg/collector"
	"github.com/kyma-project/kyma-metrics-collector/pkg/collector/edp"
	"github.com/kyma-project/kyma-metrics-collector/pkg/config"
	"github.com/kyma-project/kyma-metrics-collector/pkg/keb"
	"github.com/kyma-project/kyma-metrics-collector/pkg/queue"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
)

type Process struct {
	KEBClient             *keb.Client
	EDPClient             *edp.Client
	EDPCollector          collector.CollectorSender
	Queue                 workqueue.TypedDelayingInterface[string]
	KubeconfigProvider    runtime.ConfigProvider
	Cache                 *gocache.Cache
	PublicCloudSpecs      *config.PublicCloudSpecs
	ScrapeInterval        time.Duration
	WorkersPoolSize       int
	Logger                *zap.SugaredLogger
	ClientFactory         runtime.ClientFactory
	globalAccToBeFiltered map[string]struct{}
}

const (
	trackableTrue  = true
	trackableFalse = false
)

// New creates a new Process object.
func New(
	kebClient *keb.Client,
	edpClient *edp.Client,
	edpCollector collector.CollectorSender,
	configProvider runtime.ConfigProvider,
	publicCloudSpecs *config.PublicCloudSpecs,
	scrapeInterval time.Duration,
	workerPoolSize int,
	logger *zap.SugaredLogger,
	fileName string,
) (*Process, error) {
	switch {
	case logger == nil,
		configProvider == nil,
		publicCloudSpecs == nil,
		edpCollector == nil,
		edpClient == nil,
		kebClient == nil:
		return nil, fmt.Errorf("missing required parameter")
	}

	// Creating kubeconfigprovider with no expiration and the data will never be cleaned up
	cache := gocache.New(gocache.NoExpiration, gocache.NoExpiration)

	filterList := make(map[string]struct{})

	if fileName != "" {
		data, err := readFilterFile(fileName)
		if err != nil {
			return nil, fmt.Errorf("failed to read clusters to be filtered file: %v", err)
		}

		filterList, err = parseRuntimesToBeFiltered(data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse cluster list to filter list: %v", err)
		}
	}

	return &Process{
		KEBClient:             kebClient,
		EDPClient:             edpClient,
		EDPCollector:          edpCollector,
		KubeconfigProvider:    configProvider,
		Logger:                logger,
		PublicCloudSpecs:      publicCloudSpecs,
		Cache:                 cache,
		ScrapeInterval:        scrapeInterval,
		Queue:                 queue.NewQueue("trackable-skrs"),
		WorkersPoolSize:       workerPoolSize,
		ClientFactory:         runtime.NewClientsFactory(),
		globalAccToBeFiltered: filterList,
	}, nil
}

// Start runs the complete process of collection and sending metrics.
func (p *Process) Start() {
	var wg sync.WaitGroup

	go func() {
		p.pollKEBForRuntimes()
	}()

	for i := range p.WorkersPoolSize {
		j := i
		go func() {
			defer wg.Done()
			p.execute(j)
			p.namedLogger().Debugf("########  Worker exits ########")
		}()
	}

	wg.Wait()
}

// Execute is executed by each worker to process an entry from the queue.
func (p *Process) execute(identifier int) {
	for {
		// Pick up a subAccountID to process from queue and mark as Done()
		subAccountIDObj, _ := p.Queue.Get()
		subAccountID := fmt.Sprintf("%v", subAccountIDObj)

		// Implement cleanup holistically in #kyma-project/control-plane/issues/512
		// if isShuttingDown {
		//	//p.Cleanup()
		//	return
		// }

		requeue := p.processSubAccountID(subAccountID, identifier)
		p.Queue.Done(subAccountID)

		if requeue {
			p.Queue.AddAfter(subAccountID, p.ScrapeInterval)
		}
	}
}
