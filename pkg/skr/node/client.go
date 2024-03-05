package node

import (
	"context"
	"encoding/json"
	kmccache "github.com/kyma-project/kyma-metrics-collector/pkg/cache"

	corev1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"

	skrcommons "github.com/kyma-project/kyma-metrics-collector/pkg/skr/commons"
)

type Client struct {
	Resource  dynamic.NamespaceableResourceInterface
	ShootInfo kmccache.Record
}

func (c Config) NewClient(shootInfo kmccache.Record) (*Client, error) {
	restClientConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(shootInfo.KubeConfig))
	if err != nil {
		return nil, err
	}
	dynamicClient, err := dynamic.NewForConfig(restClientConfig)
	if err != nil {
		return nil, err
	}
	nsResourceClient := dynamicClient.Resource(GroupVersionResource())
	return &Client{Resource: nsResourceClient, ShootInfo: shootInfo}, nil
}

func (c Client) List(ctx context.Context) (*corev1.NodeList, error) {
	nodesUnstructured, err := c.Resource.Namespace(corev1.NamespaceAll).List(ctx, metaV1.ListOptions{})
	if err != nil {
		skrcommons.RecordSKRQuery(false, skrcommons.ListingNodesAction, c.ShootInfo)
		return nil, err
	}
	skrcommons.RecordSKRQuery(true, skrcommons.ListingNodesAction, c.ShootInfo)
	return convertRuntimeListToNodeList(nodesUnstructured)
}

func convertRuntimeListToNodeList(unstructuredNodesList *unstructured.UnstructuredList) (*corev1.NodeList, error) {
	nodeList := new(corev1.NodeList)
	nodeListBytes, err := unstructuredNodesList.MarshalJSON()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(nodeListBytes, nodeList)
	if err != nil {
		return nil, err
	}
	return nodeList, nil
}

func GroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Version:  corev1.SchemeGroupVersion.Version,
		Group:    corev1.SchemeGroupVersion.Group,
		Resource: "nodes",
	}
}
