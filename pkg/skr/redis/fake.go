package svc

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"

	kmccache "github.com/kyma-project/kyma-metrics-collector/pkg/cache"
	skrcommons "github.com/kyma-project/kyma-metrics-collector/pkg/skr/commons"
	kmctesting "github.com/kyma-project/kyma-metrics-collector/pkg/testing"
)

type FakeSvcClient struct{}

func (fakeSvcClient FakeSvcClient) NewClient(kmccache.Record) (*Client, error) {
	nodeList := kmctesting.GetSvcsWithLoadBalancers()
	scheme, err := skrcommons.SetupScheme()
	if err != nil {
		return nil, err
	}

	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme,
		map[schema.GroupVersionResource]string{
			{Group: "core", Version: "v1", Resource: "Node"}: "NodeList",
		}, nodeList)

	nsResourceClient := dynamicClient.Resource(GroupVersionResource())
	return &Client{Resource: nsResourceClient}, nil
}
