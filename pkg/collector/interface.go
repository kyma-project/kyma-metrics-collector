package collector

import (
	"context"

	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
)

type ScanMap map[resource.ScannerID]resource.ScanConverter

type CollectorSender interface {
	// CollectAndSend collects and sends the measures to the backend. It returns the measures collected.
	CollectAndSend(context context.Context, runtime *runtime.Info, previousScans ScanMap) (ScanMap, error)
}
