package stubs

import (
	"context"

	"github.com/kyma-project/kyma-metrics-collector/pkg/collector"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
)

type Collector struct {
	newScanMap collector.ScanMap
	err        error
}

func NewCollector(newScanMap collector.ScanMap, err error) Collector {
	return Collector{
		newScanMap: newScanMap,
		err:        err,
	}
}

func (c Collector) CollectAndSend(ctx context.Context, runtime *runtime.Info, previousScans collector.ScanMap) (collector.ScanMap, error) {
	return c.newScanMap, c.err
}
