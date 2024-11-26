package redis

import (
	"errors"
	"fmt"
	"time"

	cloudresourcesv1beta1 "github.com/kyma-project/cloud-manager/api/cloud-resources/v1beta1"

	"github.com/kyma-project/kyma-metrics-collector/pkg/process"
	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
)

var ErrRedisTierNotDefined = errors.New("Redis tier not defined")

type Scan struct {
	AWS   cloudresourcesv1beta1.AwsRedisInstanceList
	Azure cloudresourcesv1beta1.AzureRedisInstanceList
	GCP   cloudresourcesv1beta1.GcpRedisInstanceList
}

func (m *Scan) UM(duration time.Duration) (resource.UMMeasurement, error) {
	panic("implement me")
}

func (m *Scan) EDP(specs *process.PublicCloudSpecs) (resource.EDPMeasurement, error) {
	edp := resource.EDPMeasurement{}

	var errs []error

	for _, tier := range m.listTiers() {
		redisStorage := specs.GetRedisInfo(tier)
		if redisStorage == nil {
			errs = append(errs, fmt.Errorf("%w: %s", ErrRedisTierNotDefined, tier))
			continue
		}

		// Redis storage is calculated in the same way as PVC storage, but no rounding is needed
		edp.ProvisionedVolumes.SizeGbTotal += int64(redisStorage.PriceStorageGB)
		edp.ProvisionedVolumes.SizeGbRounded += int64(redisStorage.PriceStorageGB)
		edp.ProvisionedVolumes.Count++
	}

	return edp, errors.Join(errs...)
}

func (m *Scan) listTiers() []string {
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

var _ resource.ScanConverter = &Scan{}
