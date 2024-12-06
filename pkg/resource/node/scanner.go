package node

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/kyma-project/kyma-metrics-collector/pkg/config"
	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var _ resource.Scanner = &Scanner{}

type Scanner struct {
	clientFactory func(config *rest.Config) (kubernetes.Interface, error)

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
	ctx, span := otel.Tracer("").Start(ctx, "kmc.node_scan",
		trace.WithAttributes(
			attribute.String("provider", runtime.ProviderType),
			attribute.String("runtime_id", runtime.RuntimeID),
			attribute.String("subaccount_id", runtime.SubAccountID),
			attribute.String("shoot_name", runtime.ShootName),
		),
	)
	defer span.End()

	clientset, err := s.createClientset(&runtime.Kubeconfig)
	if err != nil {
		retErr := fmt.Errorf("failed to create clientset: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return nil, retErr
	}

	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		retErr := fmt.Errorf("failed to list nodes: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return nil, retErr
	}

	return &Scan{
		providerType: runtime.ProviderType,
		nodes:        *nodes,
	}, nil
}

func (s *Scanner) createClientset(config *rest.Config) (kubernetes.Interface, error) {
	if s.clientFactory == nil {
		return kubernetes.NewForConfig(config)
	}

	return s.clientFactory(config)
}
