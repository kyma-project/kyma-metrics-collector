package node

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kyma-project/kyma-metrics-collector/pkg/config"
	kmcotel "github.com/kyma-project/kyma-metrics-collector/pkg/otel"
	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
)

var _ resource.Scanner = &Scanner{}

var ErrNoNodesFound = errors.New("no nodes found")

type Scanner struct {
	specs *config.PublicCloudSpecs
}

func NewScanner(specs *config.PublicCloudSpecs) *Scanner {
	return &Scanner{
		specs: specs,
	}
}

func (s *Scanner) ID() resource.ScannerID {
	return "node"
}

func (s *Scanner) Scan(ctx context.Context, runtime *runtime.Info, clients runtime.Interface) (resource.ScanConverter, error) {
	ctx, span := otel.Tracer("").Start(ctx, "node_scan", kmcotel.SpanAttributes(runtime))
	defer span.End()

	list, err := clients.Metadata().Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"}).List(ctx, metav1.ListOptions{})
	if err != nil {
		retErr := fmt.Errorf("failed to list list: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return nil, retErr
	}

	// a cluster with no nodes is not a valid cluster
	if len(list.Items) == 0 {
		return nil, ErrNoNodesFound
	}

	return &Scan{
		providerType: runtime.ProviderType,
		specs:        s.specs,
		list:         *list,
	}, nil
}
