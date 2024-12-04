package cache

import (
	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
)

type Record struct {
	InstanceID      string
	RuntimeID       string
	SubAccountID    string
	GlobalAccountID string
	ShootName       string
	ProviderType    string
	KubeConfig      string
	Metric          *resource.EDPMeasurement
}
