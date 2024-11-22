package redis

import (
	"errors"
	"fmt"
	"time"

	cloudresourcesv1beta1 "github.com/kyma-project/cloud-manager/api/cloud-resources/v1beta1"

	"github.com/kyma-project/kyma-metrics-collector/pkg/measurer"
	"github.com/kyma-project/kyma-metrics-collector/pkg/process"
)

var (
	ErrRedisTierNotDefined = errors.New("Redis tier not defined")
)

type Measurement struct {
	AWS   cloudresourcesv1beta1.AwsRedisInstanceList
	Azure cloudresourcesv1beta1.AzureRedisInstanceList
	GCP   cloudresourcesv1beta1.GcpRedisInstanceList
}

func (m *Measurement) UM(duration time.Duration) measurer.UMData {
	// TODO implement me
	panic("implement me")
}

func (m *Measurement) EDP(specs *process.PublicCloudSpecs) (measurer.EDPData, error) {
	edp := measurer.EDPData{}

	var errs []error
	for _, tier := range m.listTiers() {
		redisStorage := specs.GetRedisInfo(tier)
		if redisStorage == nil {
			errs = append(errs, fmt.Errorf("%w: %s", tier))
			continue
		}

		// Redis storage is calculated in the same way as PVC storage, but no rounding is needed
		edp.ProvisionedVolumes.SizeGbTotal += int64(redisStorage.PriceStorageGB)
		edp.ProvisionedVolumes.SizeGbRounded += int64(redisStorage.PriceStorageGB)
		edp.ProvisionedVolumes.Count++
	}
	return edp, errors.Join(errs...)
}

func (m *Measurement) listTiers() []string {
	var tiers []string

	for _, redis := range m.AWS.Items {
		tiers = append(tiers, string(redis.Spec.RedisTier))
	}

	for _, redis := range m.Azure.Items {
		tiers = append(tiers, string(redis.Spec.RedisTier))
	}

	for _, redis := range m.GCP.Items {
		tiers = append(tiers, string(redis.Spec.RedisTier))
	}

	return tiers
}

var _ measurer.Measurement = &Measurement{}
