package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	"github.com/kyma-project/kyma-metrics-collector/pkg/config"
	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
	skrcommons "github.com/kyma-project/kyma-metrics-collector/pkg/skr/commons"
)

const (
	cloudResourcesGroup   = "cloud-resources.kyma-project.io"
	cloudResourcesVersion = "v1beta1"
)

var (
	awsRedisGVR   = schema.GroupVersionResource{Group: cloudResourcesGroup, Version: cloudResourcesVersion, Resource: "awsredisinstances"}
	azureRedisGVR = schema.GroupVersionResource{Group: cloudResourcesGroup, Version: cloudResourcesVersion, Resource: "azureredisinstances"}
	gcpRedisGVR   = schema.GroupVersionResource{Group: cloudResourcesGroup, Version: cloudResourcesVersion, Resource: "gcpredisinstances"}
)

var _ resource.Scanner = &Scanner{}

type Scanner struct {
	clientFactory func(config *rest.Config) (dynamic.Interface, error)

	specs *config.PublicCloudSpecs
}

func NewScanner(specs *config.PublicCloudSpecs) *Scanner {
	return &Scanner{
		specs: specs,
	}
}

func (s *Scanner) ID() resource.ScannerID {
	return "redis"
}

func (s *Scanner) Scan(ctx context.Context, runtime *runtime.Info) (resource.ScanConverter, error) {
	ctx, span := otel.Tracer("").Start(ctx, "kmc.redis_scan")
	defer span.End()

	dynamicClient, err := s.createClientFactory(&runtime.Kubeconfig)
	if err != nil {
		return nil, err
	}

	aws := dynamicClient.Resource(awsRedisGVR)
	azure := dynamicClient.Resource(azureRedisGVR)
	gcp := dynamicClient.Resource(gcpRedisGVR)

	scan := Scan{
		specs: s.specs,
	}

	var errs []error

	if err := listRedisInstances(ctx, aws, &scan.aws, func(success bool) {
		skrcommons.RecordSKRQuery(success, skrcommons.ListingRedisesAWSAction, runtime.ShootInfo)
	}); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		errs = append(errs, err)
	}

	if err := listRedisInstances(ctx, azure, &scan.azure, func(success bool) {
		skrcommons.RecordSKRQuery(success, skrcommons.ListingRedisesAzureAction, runtime.ShootInfo)
	}); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		errs = append(errs, err)
	}

	if err := listRedisInstances(ctx, gcp, &scan.gcp, func(success bool) {
		skrcommons.RecordSKRQuery(success, skrcommons.ListingRedisesGCPAction, runtime.ShootInfo)
	}); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		errs = append(errs, err)
	}

	if len(errs) == 0 {
		return &scan, nil
	}

	return nil, errors.Join(errs...)
}

func (s *Scanner) createClientFactory(config *rest.Config) (dynamic.Interface, error) {
	if s.clientFactory == nil {
		return dynamic.NewForConfig(config)
	}

	return s.clientFactory(config)
}

func listRedisInstances(
	ctx context.Context,
	client dynamic.NamespaceableResourceInterface,
	targetList any,
	recordMetricFn func(bool),
) error {
	unstructuredList, err := client.Namespace(corev1.NamespaceAll).List(ctx, metaV1.ListOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}

		// do not return error if CRD is not found
		recordMetricFn(false)

		return err
	}

	recordMetricFn(true)

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
