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

	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
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

type Scanner struct{}

func (m Scanner) Scan(ctx context.Context, config *rest.Config) (resource.ScanConverter, error) {
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	aws := dynamicClient.Resource(AWSRedisGVR)
	azure := dynamicClient.Resource(AzureRedisGVR)
	gcp := dynamicClient.Resource(GCPRedisGVR)

	scan := Scan{}

	var errs []error

	if err := listRedisInstances(ctx, aws, skrcommons.ListingRedisesAWSAction, &scan.AWS); err != nil {
		errs = append(errs, err)
	}

	if err := listRedisInstances(ctx, azure, skrcommons.ListingRedisesAzureAction, &scan.Azure); err != nil {
		errs = append(errs, err)
	}

	if err := listRedisInstances(ctx, gcp, skrcommons.ListingRedisesGCPAction, &scan.GCP); err != nil {
		errs = append(errs, err)
	}

	return &scan, errors.Join(errs...)
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

func (m Scanner) ID() resource.ScannerID {
	panic("implement me")
}

var _ resource.Scanner = &Scanner{}
