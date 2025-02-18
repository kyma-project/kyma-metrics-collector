package config

import (
	"fmt"
	"testing"

	"github.com/onsi/gomega"

	"github.com/kyma-project/kyma-metrics-collector/env"
)

const (
	testPublicCloudSpecsPath           = "../testing/fixtures/public_cloud_specs.json"
	testPublicCloudSpecsPathFractional = "../testing/fixtures/public_cloud_specs_fractional.json"
)

func TestGetFeature(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	config := &env.Config{PublicCloudSpecsPath: testPublicCloudSpecsPath}
	specs, err := LoadPublicCloudSpecs(config)
	g.Expect(err).Should(gomega.BeNil())

	testCases := []struct {
		cloudProvider   string
		vmType          string
		expectedFeature Feature
		wantNil         bool
	}{
		{
			cloudProvider: "azure",
			vmType:        "standard_a2_v2",
			expectedFeature: Feature{
				CpuCores: 2,
				Memory:   4,
			},
		},
		{
			cloudProvider: "azure",
			vmType:        "standard_d8_v3",
			expectedFeature: Feature{
				CpuCores: 8,
				Memory:   32,
			},
		},
		{
			cloudProvider: "azure",
			vmType:        "standard_d4s_v5",
			expectedFeature: Feature{
				CpuCores: 4,
				Memory:   16,
			},
		},
		{
			cloudProvider: "azure",
			vmType:        "standard_d8s_v5",
			expectedFeature: Feature{
				CpuCores: 8,
				Memory:   32,
			},
		},
		{
			cloudProvider: "azure",
			vmType:        "standard_d16s_v5",
			expectedFeature: Feature{
				CpuCores: 16,
				Memory:   64,
			},
		},
		{
			cloudProvider: "azure",
			vmType:        "standard_d32s_v5",
			expectedFeature: Feature{
				CpuCores: 32,
				Memory:   128,
			},
		},
		{
			cloudProvider: "azure",
			vmType:        "standard_d48s_v5",
			expectedFeature: Feature{
				CpuCores: 48,
				Memory:   192,
			},
		},
		{
			cloudProvider: "azure",
			vmType:        "standard_d64s_v5",
			expectedFeature: Feature{
				CpuCores: 64,
				Memory:   256,
			},
		},
		{
			cloudProvider: "azure",
			vmType:        "standard_d8_foo",
			wantNil:       true,
		},
		{
			cloudProvider: "aws",
			vmType:        "m5.2xlarge",
			expectedFeature: Feature{
				CpuCores: 8,
				Memory:   32,
			},
		},
		{
			cloudProvider: "aws",
			vmType:        "m5.2xlarge.foo",
			wantNil:       true,
		},
		{
			cloudProvider: "gcp",
			vmType:        "n2-standard-8",
			expectedFeature: Feature{
				CpuCores: 8,
				Memory:   32,
			},
		},
		{
			cloudProvider: "gcp",
			vmType:        "n2-standard-16",
			expectedFeature: Feature{
				CpuCores: 16,
				Memory:   64,
			},
		},
		{
			cloudProvider: "sapconvergedcloud",
			vmType:        "g_c12_m48",
			expectedFeature: Feature{
				CpuCores: 12,
				Memory:   48,
			},
		},
		{
			cloudProvider: "sapconvergedcloud",
			vmType:        "g_c16_m64",
			expectedFeature: Feature{
				CpuCores: 16,
				Memory:   64,
			},
		},
		{
			cloudProvider: "sapconvergedcloud",
			vmType:        "g_c32_m128",
			expectedFeature: Feature{
				CpuCores: 32,
				Memory:   128,
			},
		},
		{
			cloudProvider: "sapconvergedcloud",
			vmType:        "g_c4_m16",
			expectedFeature: Feature{
				CpuCores: 4,
				Memory:   16,
			},
		},
		{
			cloudProvider: "sapconvergedcloud",
			vmType:        "g_c64_m256",
			expectedFeature: Feature{
				CpuCores: 64,
				Memory:   256,
			},
		},
		{
			cloudProvider: "sapconvergedcloud",
			vmType:        "g_c6_m24",
			expectedFeature: Feature{
				CpuCores: 6,
				Memory:   24,
			},
		},
		{
			cloudProvider: "sapconvergedcloud",
			vmType:        "g_c8_m32",
			expectedFeature: Feature{
				CpuCores: 8,
				Memory:   32,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s-%s", tc.cloudProvider, tc.vmType), func(t *testing.T) {
			gotFeature := specs.GetFeature(tc.cloudProvider, tc.vmType)
			if tc.wantNil {
				g.Expect(gotFeature).Should(gomega.BeNil())
				return
			}

			g.Expect(*gotFeature).Should(gomega.Equal(tc.expectedFeature))
		})
	}
}

func TestGetFeatureFractional(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	config := &env.Config{PublicCloudSpecsPath: testPublicCloudSpecsPathFractional}
	specs, err := LoadPublicCloudSpecs(config)
	g.Expect(err).Should(gomega.BeNil())

	testCases := []struct {
		cloudProvider   string
		vmType          string
		expectedFeature Feature
		wantNil         bool
	}{
		{
			cloudProvider: "azure",
			vmType:        "standard_a1_v2",
			expectedFeature: Feature{
				CpuCores: 1.1,
				Memory:   2.1,
			},
		},
		{
			cloudProvider: "aws",
			vmType:        "m4.large",
			expectedFeature: Feature{
				CpuCores: 2.2,
				Memory:   8.2,
			},
		},
		{
			cloudProvider: "gcp",
			vmType:        "n1-standard-4",
			expectedFeature: Feature{
				CpuCores: 4.3,
				Memory:   15.3,
			},
		},
		{
			cloudProvider: "sapconvergedcloud",
			vmType:        "g_c12_m48",
			expectedFeature: Feature{
				CpuCores: 12.4,
				Memory:   48.4,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s-%s", tc.cloudProvider, tc.vmType), func(t *testing.T) {
			gotFeature := specs.GetFeature(tc.cloudProvider, tc.vmType)
			if tc.wantNil {
				g.Expect(gotFeature).Should(gomega.BeNil())
				return
			}

			g.Expect(*gotFeature).Should(gomega.Equal(tc.expectedFeature))
		})
	}
}

func TestGetRedisInfo(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	config := &env.Config{PublicCloudSpecsPath: testPublicCloudSpecsPath}
	specs, err := LoadPublicCloudSpecs(config)
	g.Expect(err).Should(gomega.BeNil())

	testCases := []struct {
		tier          string
		expectedRedis RedisInfo
		wantNil       bool
	}{
		{
			tier: "S1",
			expectedRedis: RedisInfo{
				PriceStorageGB:     182,
				PriceCapacityUnits: 74,
			},
		},
		{
			tier: "P1",
			expectedRedis: RedisInfo{
				PriceStorageGB:     1903,
				PriceCapacityUnits: 773,
			},
		},
		{
			tier:    "foo",
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.tier, func(t *testing.T) {
			gotRedis := specs.GetRedisInfo(tc.tier)
			if tc.wantNil {
				g.Expect(gotRedis).Should(gomega.BeNil())
				return
			}

			g.Expect(gotRedis).Should(gomega.Not(gomega.BeNil()))
			g.Expect(*gotRedis).Should(gomega.Equal(tc.expectedRedis))
		})
	}
}

func TestGetRedisInfoFractional(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	config := &env.Config{PublicCloudSpecsPath: testPublicCloudSpecsPathFractional}
	specs, err := LoadPublicCloudSpecs(config)
	g.Expect(err).Should(gomega.BeNil())

	testCases := []struct {
		tier          string
		expectedRedis RedisInfo
		wantNil       bool
	}{
		{
			tier: "S1",
			expectedRedis: RedisInfo{
				PriceStorageGB:     182.5,
				PriceCapacityUnits: 74.5,
			},
		},
		{
			tier:    "foo",
			wantNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.tier, func(t *testing.T) {
			gotRedis := specs.GetRedisInfo(tc.tier)
			if tc.wantNil {
				g.Expect(gotRedis).Should(gomega.BeNil())
				return
			}

			g.Expect(gotRedis).Should(gomega.Not(gomega.BeNil()))
			g.Expect(*gotRedis).Should(gomega.Equal(tc.expectedRedis))
		})
	}
}
