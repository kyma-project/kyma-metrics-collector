package cache

import "github.com/kyma-project/kyma-metrics-collector/pkg/edp"

type Record struct {
	InstanceID      string
	RuntimeID       string
	SubAccountID    string
	GlobalAccountID string
	ShootName       string
	ProviderType    string
	KubeConfig      string
	Metric          *edp.ConsumptionMetrics
}
