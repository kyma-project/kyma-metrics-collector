package collector

import (
	"context"

	"k8s.io/client-go/rest"

	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
)

type ScanMap map[resource.ScannerID]resource.ScanConverter

type CollectorSender interface {
	// CollectAndSend collects and sends the measures to the backend. It returns the measures collected.
	CollectAndSend(context context.Context, config *rest.Config, previousScans ScanMap) (ScanMap, error)
}
