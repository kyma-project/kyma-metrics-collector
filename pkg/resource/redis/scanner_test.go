package redis

import (
	"context"
	"errors"
	"testing"

	cloudresourcesv1beta1 "github.com/kyma-project/cloud-manager/api/cloud-resources/v1beta1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"

	"github.com/kyma-project/kyma-metrics-collector/pkg/config"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
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
	clientFactory := func(*rest.Config) (dynamic.Interface, error) {
		scheme := k8sruntime.NewScheme()
		if err := cloudresourcesv1beta1.AddToScheme(scheme); err != nil {
			return nil, err
		}

		dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme,
			map[schema.GroupVersionResource]string{
				awsRedisGVR:   "AwsRedisInstanceList",
				azureRedisGVR: "AzureRedisInstanceList",
				gcpRedisGVR:   "GcpRedisInstanceList",
			}, awsRedises)
		return dynamicClient, nil
	}

	scanner := Scanner{
		clientFactory: clientFactory,
		specs:         &config.PublicCloudSpecs{},
	}

	provider := "test-provider"
	result, err := scanner.Scan(context.Background(), &runtime.Info{
		ProviderType: provider,
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	redisScan, ok := result.(*Scan)
	require.True(t, ok)
	require.Equal(t, awsRedises.Items, redisScan.aws.Items)
	require.Equal(t, scanner.specs, redisScan.specs)
}

func TestScanner_Scan_Error(t *testing.T) {
	clientFactory := func(*rest.Config) (dynamic.Interface, error) {
		scheme := k8sruntime.NewScheme()
		if err := cloudresourcesv1beta1.AddToScheme(scheme); err != nil {
			return nil, err
		}

		dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)
		dynamicClient.PrependReactor("list", "awsredisinstances", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
			return true, nil, errors.New("failed to list aws redis instances")
		})
		return dynamicClient, nil
	}

	scanner := Scanner{
		clientFactory: clientFactory,
	}
	result, err := scanner.Scan(context.Background(), &runtime.Info{})

	require.Error(t, err)
	require.Nil(t, result)
}
