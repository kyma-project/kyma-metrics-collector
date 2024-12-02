package redis

import (
	"context"
	"errors"
	"strconv"
	"testing"

	cloudresourcesv1beta1 "github.com/kyma-project/cloud-manager/api/cloud-resources/v1beta1"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"

	kmccache "github.com/kyma-project/kyma-metrics-collector/pkg/cache"
	"github.com/kyma-project/kyma-metrics-collector/pkg/config"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
	skrcommons "github.com/kyma-project/kyma-metrics-collector/pkg/skr/commons"
)

var fakeShootInfo = kmccache.Record{
	InstanceID:      "adccb200-6052-4192-8adf-785b8a5af306",
	RuntimeID:       "fe5ab5d6-5b0b-4b70-9644-7f89d230b516",
	SubAccountID:    "1ae0dbe1-d13d-4e39-bed4-7c83364084d5",
	GlobalAccountID: "0c22f798-e572-4fc7-a502-cd825c742ff6",
	ShootName:       "c-987654",
}

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
		ShootInfo:    fakeShootInfo,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	redisScan, ok := result.(*Scan)
	require.True(t, ok)
	require.Equal(t, awsRedises.Items, redisScan.aws.Items)
	require.Equal(t, scanner.specs, redisScan.specs)

	for _, actions := range []string{
		skrcommons.ListingRedisesAWSAction,
		skrcommons.ListingRedisesAzureAction,
		skrcommons.ListingRedisesGCPAction,
	} {
		gotMetrics, err := skrcommons.TotalQueriesMetric.GetMetricWithLabelValues(
			actions,
			strconv.FormatBool(true),
			fakeShootInfo.ShootName,
			fakeShootInfo.InstanceID,
			fakeShootInfo.RuntimeID,
			fakeShootInfo.SubAccountID,
			fakeShootInfo.GlobalAccountID,
		)
		require.NoError(t, err)
		require.Equal(t, 1, int(testutil.ToFloat64(gotMetrics)))
	}
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

	result, err := scanner.Scan(context.Background(), &runtime.Info{ShootInfo: fakeShootInfo})

	require.Error(t, err)
	require.Nil(t, result)

	gotMetrics, err := skrcommons.TotalQueriesMetric.GetMetricWithLabelValues(
		skrcommons.ListingRedisesAWSAction,
		strconv.FormatBool(false),
		fakeShootInfo.ShootName,
		fakeShootInfo.InstanceID,
		fakeShootInfo.RuntimeID,
		fakeShootInfo.SubAccountID,
		fakeShootInfo.GlobalAccountID,
	)
	require.NoError(t, err)
	require.Equal(t, 1, int(testutil.ToFloat64(gotMetrics)))
}
