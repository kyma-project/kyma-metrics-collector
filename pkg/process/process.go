package process

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/workqueue"

	"github.com/kyma-project/kyma-metrics-collector/pkg/collector"
	"github.com/kyma-project/kyma-metrics-collector/pkg/collector/edp"
	"github.com/kyma-project/kyma-metrics-collector/pkg/config"
	"github.com/kyma-project/kyma-metrics-collector/pkg/keb"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
)

type Process struct {
	KEBClient          *keb.Client
	EDPClient          *edp.Client
	EDPCollector       collector.CollectorSender
	Queue              workqueue.TypedDelayingInterface[string]
	KubeconfigProvider runtime.ConfigProvider
	Cache              *cache.Cache
	PublicCloudSpecs   *config.PublicCloudSpecs
	ScrapeInterval     time.Duration
	WorkersPoolSize    int
	Logger             *zap.SugaredLogger
	ClientFactory      runtime.ClientFactory
}

const (
	trackableTrue  = true
	trackableFalse = false
)

type HttpClientFactory interface {
	NewClient(config *rest.Config) (*http.Client, error)
}

var _ HttpClientFactory = &K8sClientFactory{}

type K8sClientFactory struct{}

func (f *K8sClientFactory) NewClient(config *rest.Config) (*http.Client, error) {
	// Create HTTP client from REST client config. Use proxy from environment
	// setting the proxy to http.ProxyFromEnvironment will avoid the client to cache the TLSConfiguration. This is important as it leads to a memory leak.
	// Scanners have to use the same client to avoid the memory leak.
	// After all scanners are done, all connections opened by the client will be closed.
	// See: https://github.com/kubernetes/kubernetes/issues/109289
	config.Proxy = http.ProxyFromEnvironment

	client, err := rest.HTTPClientFor(config)
	if err != nil {
		return nil, err
	}

	return client, nil
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
