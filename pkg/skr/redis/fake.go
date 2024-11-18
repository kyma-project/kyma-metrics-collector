package redis

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"

	kmccache "github.com/kyma-project/kyma-metrics-collector/pkg/cache"
	skrcommons "github.com/kyma-project/kyma-metrics-collector/pkg/skr/commons"
	kmctesting "github.com/kyma-project/kyma-metrics-collector/pkg/testing"
)

type FakeRedisClient struct{}

func (fakeSvcClient FakeRedisClient) NewClient(kmccache.Record) (*Client, error) {
	nodeList := kmctesting.GetSvcsWithLoadBalancers()

	scheme, err := skrcommons.SetupScheme()
	if err != nil {
		return nil, err
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme,
		map[schema.GroupVersionResource]string{
			AWSRedisGVR:   "AwsRedisInstanceList",
			AzureRedisGVR: "AzureRedisInstanceList",
			GCPRedisGVR:   "GcpRedisInstanceList",
		}, nodeList)

	return &Client{
		AWSRedisClient:   dynamicClient.Resource(AWSRedisGVR),
		AzureRedisClient: dynamicClient.Resource(AzureRedisGVR),
		GCPRedisClient:   dynamicClient.Resource(GCPRedisGVR),
	}, nil
}
