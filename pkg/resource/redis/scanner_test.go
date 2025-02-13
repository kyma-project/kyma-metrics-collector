package redis

import (
	"errors"
	"testing"

	cloudresourcesv1beta1 "github.com/kyma-project/cloud-manager/api/cloud-resources/v1beta1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/kyma-project/kyma-metrics-collector/pkg/config"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime/stubs"
)

func TestScanner_ID(t *testing.T) {
	scanner := Scanner{}
	require.Equal(t, "redis", string(scanner.ID()), "Scanner ID should be 'redis'")
}

func TestScanner_Scan_Successful(t *testing.T) {
	awsRedises := &cloudresourcesv1beta1.AwsRedisInstanceList{
		Items: []cloudresourcesv1beta1.AwsRedisInstance{
			{ObjectMeta: metav1.ObjectMeta{Name: "aws-redis-1"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "aws-redis-2"}},
		},
	}

	scheme := k8sruntime.NewScheme()
	err := cloudresourcesv1beta1.AddToScheme(scheme)
	require.NoError(t, err)

	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme,
		map[schema.GroupVersionResource]string{
			awsRedisGVR:   "AwsRedisInstanceList",
			azureRedisGVR: "AzureRedisInstanceList",
			gcpRedisGVR:   "GcpRedisInstanceList",
		}, awsRedises)

	clients := stubs.Clients{
		DynamicInterface: dynamicClient,
	}

	scanner := Scanner{
		specs: &config.PublicCloudSpecs{},
	}

	provider := "test-provider"

	result, err := scanner.Scan(t.Context(), &runtime.Info{
		ProviderType: provider,
	}, clients)
	require.NoError(t, err)
	require.NotNil(t, result)

	redisScan, ok := result.(*Scan)
	require.True(t, ok)
	require.Equal(t, awsRedises.Items, redisScan.aws.Items)
	require.Equal(t, scanner.specs, redisScan.specs)
}

func TestScanner_Scan_Error(t *testing.T) {
	scheme := k8sruntime.NewScheme()
	err := cloudresourcesv1beta1.AddToScheme(scheme)
	require.NoError(t, err)

	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)
	dynamicClient.PrependReactor("list", "awsredisinstances", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
		return true, nil, errors.New("failed to list aws redis instances")
	})

	clients := stubs.Clients{
		DynamicInterface: dynamicClient,
	}

	scanner := Scanner{}

	result, err := scanner.Scan(t.Context(), &runtime.Info{}, clients)

	require.Error(t, err)
	require.Nil(t, result)
}
