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

	"github.com/kyma-project/kyma-metrics-collector/pkg/config"
	kmcotel "github.com/kyma-project/kyma-metrics-collector/pkg/otel"
	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
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

func (s *Scanner) Scan(ctx context.Context, runtime *runtime.Info, clients runtime.Interface) (resource.ScanConverter, error) {
	ctx, span := otel.Tracer("").Start(ctx, "redis_scan", kmcotel.SpanAttributes(runtime))
	defer span.End()

	dynamicClient := clients.Dynamic()
	aws := dynamicClient.Resource(awsRedisGVR)
	azure := dynamicClient.Resource(azureRedisGVR)
	gcp := dynamicClient.Resource(gcpRedisGVR)

	scan := Scan{
		specs: s.specs,
	}

	var errs []error

	if err := listRedisInstances(ctx, aws, any(&scan.aws)); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		errs = append(errs, err)
	}

	if err := listRedisInstances(ctx, azure, any(&scan.azure)); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		errs = append(errs, err)
	}

	if err := listRedisInstances(ctx, gcp, any(&scan.gcp)); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		errs = append(errs, err)
	}

	if len(errs) == 0 {
		return &scan, nil
	}

	return nil, errors.Join(errs...)
}

func listRedisInstances(
	ctx context.Context,
	client dynamic.NamespaceableResourceInterface,
	targetList any,
) error {
	unstructuredList, err := client.Namespace(corev1.NamespaceAll).List(ctx, metaV1.ListOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}

		return err
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
