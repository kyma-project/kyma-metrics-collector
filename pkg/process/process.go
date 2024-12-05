package process

import (
	"fmt"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/util/workqueue"

	"github.com/kyma-project/kyma-metrics-collector/pkg/collector"
	"github.com/kyma-project/kyma-metrics-collector/pkg/collector/edp"
	"github.com/kyma-project/kyma-metrics-collector/pkg/config"
	"github.com/kyma-project/kyma-metrics-collector/pkg/keb"
	skrnode "github.com/kyma-project/kyma-metrics-collector/pkg/skr/node"
	skrpvc "github.com/kyma-project/kyma-metrics-collector/pkg/skr/pvc"
	skrredis "github.com/kyma-project/kyma-metrics-collector/pkg/skr/redis"
	skrsvc "github.com/kyma-project/kyma-metrics-collector/pkg/skr/svc"
)

type Process struct {
	KEBClient         *keb.Client
	EDPClient         *edp.Client
	EDPCollector      collector.CollectorSender
	Queue             workqueue.TypedDelayingInterface[string]
	SecretCacheClient v1.CoreV1Interface
	Cache             *cache.Cache
	PublicCloudSpecs  *config.PublicCloudSpecs
	ScrapeInterval    time.Duration
	WorkersPoolSize   int
	NodeConfig        skrnode.ConfigInf
	PVCConfig         skrpvc.ConfigInf
	SvcConfig         skrsvc.ConfigInf
	RedisConfig       skrredis.ConfigInf
	Logger            *zap.SugaredLogger
}

const (
	trackableTrue  = true
	trackableFalse = false
)

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

		p.processSubAccountID(subAccountID, identifier)
		p.Queue.Done(subAccountIDObj)
	}
}
