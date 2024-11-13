package process

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pkg/errors"

	"github.com/kyma-project/kyma-metrics-collector/env"
)

type PublicCloudSpecs struct {
	Providers Providers            `json:"providers"`
	Redis     map[string]RedisInfo `json:"redis_tiers"`
}

type Providers struct {
	Azure     map[string]Feature `json:"azure"`
	AWS       map[string]Feature `json:"aws"`
	GCP       map[string]Feature `json:"gcp"`
	OpenStack map[string]Feature `json:"sapconvergedcloud"`
}

type Feature struct {
	CpuCores int     `json:"cpu_cores"`
	Memory   float64 `json:"memory"`
	Storage  int     `json:"storage,omitempty"`
	MaxNICs  int     `json:"max_nics,omitempty"`
}

type RedisInfo struct {
	PriceStorageGB     int `json:"price_storage_gb"`
	PriceCapacityUnits int `json:"price_cu"`
}

func (pcs *PublicCloudSpecs) GetFeature(cloudProvider, vmType string) *Feature {
	switch cloudProvider {
	case AWS:
		if feature, ok := pcs.Providers.AWS[vmType]; ok {
			return &feature
		}
	case Azure:
		if feature, ok := pcs.Providers.Azure[vmType]; ok {
			return &feature
		}
	case GCP:
		if feature, ok := pcs.Providers.GCP[vmType]; ok {
			return &feature
		}
	case CCEE:
		if feature, ok := pcs.Providers.OpenStack[vmType]; ok {
			return &feature
		}
	}
	return nil
}

func (pcs *PublicCloudSpecs) GetRedisInfo(tier string) *RedisInfo {
	if redisInfo, ok := pcs.Redis[tier]; ok {
		return &redisInfo
	}

	return nil
}

// LoadPublicCloudSpecs loads string data to Providers object from an env var.
func LoadPublicCloudSpecs(cfg *env.Config) (*PublicCloudSpecs, error) {
	if cfg.PublicCloudSpecsPath == "" {
		return nil, fmt.Errorf("public cloud specification path is not configured")
	}

	specsJSON, err := os.ReadFile(cfg.PublicCloudSpecsPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read public cloud specs file")
	}

	var specs PublicCloudSpecs
	if err = json.Unmarshal(specsJSON, &specs); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal public cloud specs")
	}

	return &specs, nil
}
