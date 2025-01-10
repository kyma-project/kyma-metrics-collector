package cache

import "github.com/kyma-project/kyma-metrics-collector/pkg/collector"

type Record struct {
	InstanceID      string
	RuntimeID       string
	SubAccountID    string
	GlobalAccountID string
	ShootName       string
	ProviderType    string
	ScanMap         collector.ScanMap
}
