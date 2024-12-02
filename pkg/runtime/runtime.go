package runtime

import (
	"k8s.io/client-go/rest"

	kmcache "github.com/kyma-project/kyma-metrics-collector/pkg/cache"
)

type Info struct {
	Kubeconfig   rest.Config
	ProviderType string
	ShootInfo    kmcache.Record
}
