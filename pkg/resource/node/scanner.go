package node

import (
	"context"
	"fmt"

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
	clientset, err := kubernetes.NewForConfig(&runtime.Kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	nodes, err := clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	return &Scan{
		provider: runtime.ProviderType,
		nodes:    *nodes,
	}, nil
}
