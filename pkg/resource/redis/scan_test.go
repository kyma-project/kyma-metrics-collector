package redis

import (
	"errors"
	"testing"

	cloudresourcesv1beta1 "github.com/kyma-project/cloud-manager/api/cloud-resources/v1beta1"
	"github.com/kyma-project/kyma-metrics-collector/pkg/config"
	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
	"github.com/stretchr/testify/assert"
)

func TestScan_EDP(t *testing.T) {
	specs := &config.PublicCloudSpecs{
		Redis: map[string]config.RedisInfo{
			"s1": {PriceStorageGB: 10},
			"p1": {PriceStorageGB: 50},
		},
	}

	tests := []struct {
		name          string
		awsRedis      cloudresourcesv1beta1.AwsRedisInstanceList
		azureRedis    cloudresourcesv1beta1.AzureRedisInstanceList
		gcpRedis      cloudresourcesv1beta1.GcpRedisInstanceList
		expected      resource.EDPMeasurement
		expectedError error
	}{
		{
			name: "no redis instances",
			expected: resource.EDPMeasurement{
				ProvisionedVolumes: resource.ProvisionedVolumes{
					SizeGbTotal:   0,
					SizeGbRounded: 0,
					Count:         0,
				},
			},
			expectedError: nil,
		},
		{
			name: "single aws redis instance with valid tier",
			awsRedis: cloudresourcesv1beta1.AwsRedisInstanceList{
				Items: []cloudresourcesv1beta1.AwsRedisInstance{
					{Spec: cloudresourcesv1beta1.AwsRedisInstanceSpec{RedisTier: "s1"}},
				},
			},
			expected: resource.EDPMeasurement{
				ProvisionedVolumes: resource.ProvisionedVolumes{
					SizeGbTotal:   10,
					SizeGbRounded: 10,
					Count:         1,
				},
			},
			expectedError: nil,
		},
		{
			name: "single azure redis instance with valid tier",
			azureRedis: cloudresourcesv1beta1.AzureRedisInstanceList{
				Items: []cloudresourcesv1beta1.AzureRedisInstance{
					{Spec: cloudresourcesv1beta1.AzureRedisInstanceSpec{RedisTier: "p1"}},
				},
			},
			expected: resource.EDPMeasurement{
				ProvisionedVolumes: resource.ProvisionedVolumes{
					SizeGbTotal:   50,
					SizeGbRounded: 50, // unlike pvc, no rounding is done for redis storage
					Count:         1,
				},
			},
			expectedError: nil,
		},
		{
			name: "single gcp redis instance with valid tier",
			gcpRedis: cloudresourcesv1beta1.GcpRedisInstanceList{
				Items: []cloudresourcesv1beta1.GcpRedisInstance{
					{Spec: cloudresourcesv1beta1.GcpRedisInstanceSpec{RedisTier: "s1"}},
				},
			},
			expected: resource.EDPMeasurement{
				ProvisionedVolumes: resource.ProvisionedVolumes{
					SizeGbTotal:   10,
					SizeGbRounded: 10,
					Count:         1,
				},
			},
			expectedError: nil,
		},
		{
			name: "single azure redis instance with invalid tier",
			azureRedis: cloudresourcesv1beta1.AzureRedisInstanceList{
				Items: []cloudresourcesv1beta1.AzureRedisInstance{
					{Spec: cloudresourcesv1beta1.AzureRedisInstanceSpec{RedisTier: "invalid-tier"}},
				},
			},
			expected: resource.EDPMeasurement{
				ProvisionedVolumes: resource.ProvisionedVolumes{
					SizeGbTotal:   0,
					SizeGbRounded: 0,
					Count:         0,
				},
			},
			expectedError: ErrUnknownRedisTier,
		},
		{
			name: "mixed redis instances with valid and invalid tiers",
			awsRedis: cloudresourcesv1beta1.AwsRedisInstanceList{
				Items: []cloudresourcesv1beta1.AwsRedisInstance{
					{Spec: cloudresourcesv1beta1.AwsRedisInstanceSpec{RedisTier: "s1"}},
				},
			},
			azureRedis: cloudresourcesv1beta1.AzureRedisInstanceList{
				Items: []cloudresourcesv1beta1.AzureRedisInstance{
					{Spec: cloudresourcesv1beta1.AzureRedisInstanceSpec{RedisTier: "invalid-tier"}},
				},
			},
			expected: resource.EDPMeasurement{
				ProvisionedVolumes: resource.ProvisionedVolumes{
					SizeGbTotal:   10,
					SizeGbRounded: 10,
					Count:         1,
				},
			},
			expectedError: ErrUnknownRedisTier,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scan := &Scan{
				specs: specs,
				aws:   test.awsRedis,
				azure: test.azureRedis,
				gcp:   test.gcpRedis,
			}

			actualEDP, err := scan.EDP()

			// Validate EDP output
			assert.Equal(t, test.expected.ProvisionedVolumes, actualEDP.ProvisionedVolumes, "EDPMeasurement mismatch")

			// Validate error
			if test.expectedError != nil {
				assert.True(t, errors.Is(err, test.expectedError), "unexpected error: got %v, want %v", err, test.expectedError)
			} else {
				assert.NoError(t, err, "unexpected error: got %v", err)
			}
		})
	}
}
