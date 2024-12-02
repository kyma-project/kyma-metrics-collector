package redis

import (
	"errors"
	"fmt"
	"time"

	cloudresourcesv1beta1 "github.com/kyma-project/cloud-manager/api/cloud-resources/v1beta1"

	"github.com/kyma-project/kyma-metrics-collector/pkg/config"
	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
)

var ErrUnknownRedisTier = errors.New("Redis tier not defined")

var _ resource.ScanConverter = &Scan{}

type Scan struct {
	specs *config.PublicCloudSpecs

	aws   cloudresourcesv1beta1.AwsRedisInstanceList
	azure cloudresourcesv1beta1.AzureRedisInstanceList
	gcp   cloudresourcesv1beta1.GcpRedisInstanceList
}

func (s *Scan) UM(duration time.Duration) (resource.UMMeasurement, error) {
	return resource.UMMeasurement{}, nil
}

func (s *Scan) EDP() (resource.EDPMeasurement, error) {
	edp := resource.EDPMeasurement{}

	var errs []error

	for _, tier := range s.listTiers() {
		redisStorage := s.specs.GetRedisInfo(tier)
		if redisStorage == nil {
			errs = append(errs, fmt.Errorf("%w: %s", ErrUnknownRedisTier, tier))
			continue
		}

		// Redis storage is calculated in the same way as PVC storage, but no rounding is needed
		edp.ProvisionedVolumes.SizeGbTotal += int64(redisStorage.PriceStorageGB)
		edp.ProvisionedVolumes.SizeGbRounded += int64(redisStorage.PriceStorageGB)
		edp.ProvisionedVolumes.Count++
	}

	return edp, errors.Join(errs...)
}

func (s *Scan) listTiers() []string {
	var tiers []string

	for _, redis := range s.aws.Items {
		tiers = append(tiers, string(redis.Spec.RedisTier))
	}

	for _, redis := range s.azure.Items {
		tiers = append(tiers, string(redis.Spec.RedisTier))
	}

	for _, redis := range s.gcp.Items {
		tiers = append(tiers, string(redis.Spec.RedisTier))
	}

	return tiers
}
