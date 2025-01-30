package node

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/rest"

	"github.com/kyma-project/kyma-metrics-collector/pkg/config"
	kmcotel "github.com/kyma-project/kyma-metrics-collector/pkg/otel"
	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
)

var _ resource.Scanner = &Scanner{}

var ErrNoNodesFound = errors.New("no nodes found")

type Scanner struct {
	clientFactory func(config *rest.Config) (metadata.Interface, error)

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

func (s *Scanner) Scan(ctx context.Context, runtime *runtime.Info) (resource.ScanConverter, error) {
	ctx, span := otel.Tracer("").Start(ctx, "node_scan", kmcotel.SpanAttributes(runtime))
	defer span.End()

	cl, err := s.createClientset(&runtime.Kubeconfig, runtime.Client)
	if err != nil {
		retErr := fmt.Errorf("failed to create clientset: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return nil, retErr
	}

	list, err := cl.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"}).List(ctx, metav1.ListOptions{})
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

func (s *Scanner) createClientset(config *rest.Config, client *http.Client) (metadata.Interface, error) {
	if s.clientFactory == nil {
		return metadata.NewForConfigAndClient(config, client)
	}

	return s.clientFactory(config)
}
