package redis

import (
	"context"
	"encoding/json"
	"fmt"

	cloudresourcesv1beta1 "github.com/kyma-project/cloud-manager/api/cloud-resources/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"

	kmccache "github.com/kyma-project/kyma-metrics-collector/pkg/cache"
	skrcommons "github.com/kyma-project/kyma-metrics-collector/pkg/skr/commons"
)

const (
	cloudResourcesGroup   = "cloud-resources.kyma-project.io"
	cloudResourcesVersion = "v1beta1"
)

var (
	AWSRedisGVR   = schema.GroupVersionResource{Group: cloudResourcesGroup, Version: cloudResourcesVersion, Resource: "awsredisinstances"}
	AzureRedisGVR = schema.GroupVersionResource{Group: cloudResourcesGroup, Version: cloudResourcesVersion, Resource: "azureredisinstances"}
	GCPRedisGVR   = schema.GroupVersionResource{Group: cloudResourcesGroup, Version: cloudResourcesVersion, Resource: "gcpredisinstances"}
)

type Client struct {
	AWSRedisClient   dynamic.NamespaceableResourceInterface
	AzureRedisClient dynamic.NamespaceableResourceInterface
	GCPRedisClient   dynamic.NamespaceableResourceInterface
	ShootInfo        kmccache.Record
}

type RedisList struct {
	AWS   cloudresourcesv1beta1.AwsRedisInstanceList
	Azure cloudresourcesv1beta1.AzureRedisInstanceList
	GCP   cloudresourcesv1beta1.GcpRedisInstanceList
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

	return &Client{
		AWSRedisClient:   dynamicClient.Resource(AWSRedisGVR),
		AzureRedisClient: dynamicClient.Resource(AzureRedisGVR),
		GCPRedisClient:   dynamicClient.Resource(GCPRedisGVR),
		ShootInfo:        shootInfo,
	}, nil
}

func (c Client) List(ctx context.Context) (*RedisList, error) {
	var awsRedises cloudresourcesv1beta1.AwsRedisInstanceList
	if err := c.listRedisInstances(ctx, c.AWSRedisClient, skrcommons.ListingRedisesAWSAction, &awsRedises); err != nil {
		return nil, fmt.Errorf("failed to list aws redis instances: %w", err)
	}

	var azureRedises cloudresourcesv1beta1.AzureRedisInstanceList
	if err := c.listRedisInstances(ctx, c.AzureRedisClient, skrcommons.ListingRedisesAzureAction, &azureRedises); err != nil {
		return nil, fmt.Errorf("failed to list azure redis instances: %w", err)
	}

	var gcpRedises cloudresourcesv1beta1.GcpRedisInstanceList
	if err := c.listRedisInstances(ctx, c.GCPRedisClient, skrcommons.ListingRedisesGCPAction, &gcpRedises); err != nil {
		return nil, fmt.Errorf("failed to list gcp redis instances: %w", err)
	}

	return &RedisList{
		AWS:   awsRedises,
		Azure: azureRedises,
		GCP:   gcpRedises,
	}, nil
}

func (c Client) listRedisInstances(
	ctx context.Context,
	client dynamic.NamespaceableResourceInterface,
	actionPromLabel string,
	targetList any,
) error {
	unstructuredList, err := client.Namespace(corev1.NamespaceAll).List(ctx, metaV1.ListOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}

		skrcommons.RecordSKRQuery(false, actionPromLabel, c.ShootInfo)
		return err
	}

	skrcommons.RecordSKRQuery(true, actionPromLabel, c.ShootInfo)

	if err := convertUnstructuredListToRedisList(unstructuredList, targetList); err != nil {
		return err
	}

	return nil
}

func convertUnstructuredListToRedisList(unstructuredList *unstructured.UnstructuredList, targetList any) error {
	redisListBytes, err := unstructuredList.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal unstructured list: %w", err)
	}

	err = json.Unmarshal(redisListBytes, targetList)
	if err != nil {
		return fmt.Errorf("failed to unmarshal unstructured list: %w", err)
	}

	return nil
}
