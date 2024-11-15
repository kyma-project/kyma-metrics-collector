package node

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"

	kmccache "github.com/kyma-project/kyma-metrics-collector/pkg/cache"
	skrcommons "github.com/kyma-project/kyma-metrics-collector/pkg/skr/commons"
	kmctesting "github.com/kyma-project/kyma-metrics-collector/pkg/testing"
)

type FakeNodeClient struct{}

func (fakeNodeClient FakeNodeClient) NewClient(kmccache.Record) (*Client, error) {
	nodeList := kmctesting.Get3NodesWithStandardD8v3VMType()

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
