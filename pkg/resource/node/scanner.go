package node

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
)

var _ resource.Scanner = &Scanner{}

type Scanner struct{}

func (s Scanner) ID() resource.ScannerID {
	return "node"
}

func (s Scanner) Scan(ctx context.Context, runtime *runtime.Info) (resource.ScanConverter, error) {
	ctx, span := otel.Tracer("").Start(ctx, "kmc.node_scan")
	defer span.End()

	clientset, err := kubernetes.NewForConfig(&runtime.Kubeconfig)
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
		provider: runtime.ProviderType,
		nodes:    *nodes,
	}, nil
}
