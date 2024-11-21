package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	"github.com/kyma-project/kyma-metrics-collector/pkg/measurer"
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

type Measurer struct {
}

type client struct {
	aws, azure, gcp dynamic.NamespaceableResourceInterface
}

func (m Measurer) Measure(ctx context.Context, config *rest.Config) (measurer.Measurement, error) {
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	c := client{
		aws:   dynamicClient.Resource(AWSRedisGVR),
		azure: dynamicClient.Resource(AzureRedisGVR),
		gcp:   dynamicClient.Resource(GCPRedisGVR),
	}

	msrmnt := Measurement{}
	var errs []error
	if err := listRedisInstances(ctx, c.aws, skrcommons.ListingRedisesAWSAction, &msrmnt.AWSRedises); err != nil {
		errs = append(errs, err)
	}

	if err := listRedisInstances(ctx, c.azure, skrcommons.ListingRedisesAzureAction, &msrmnt.AzureRedises); err != nil {
		errs = append(errs, err)
	}

	if err := listRedisInstances(ctx, c.gcp, skrcommons.ListingRedisesGCPAction, &msrmnt.GCPRedises); err != nil {
		errs = append(errs, err)
	}

	return msrmnt, errors.Join(errs...)

}

func listRedisInstances(
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
	}

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

func (m Measurer) ID() measurer.MeasurerID {
	// TODO implement me
	panic("implement me")
}

var _ measurer.Measurer = &Measurer{}
