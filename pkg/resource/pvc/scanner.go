package pvc

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
)

var _ resource.Scanner = &Scanner{}

type Scanner struct {
	clientFactory func(config *rest.Config) (kubernetes.Interface, error)
}

// NewScanner creates a new instance of Scanner.
// While not strictly necessary, this factory function is provided
// for consistency with other scanner implementations.
func NewScanner() *Scanner {
	return &Scanner{}
}

func (s *Scanner) ID() resource.ScannerID {
	return "pvc"
}

func (s *Scanner) Scan(ctx context.Context, runtime *runtime.Info) (resource.ScanConverter, error) {
	ctx, span := otel.Tracer("").Start(ctx, "kmc.pvc_scan",
		trace.WithAttributes(
			attribute.String("provider", runtime.ProviderType),
			attribute.String("runtime_id", runtime.RuntimeID),
			attribute.String("sub_account_id", runtime.SubAccountID),
			attribute.String("shoot_name", runtime.ShootName),
		),
	)
	defer span.End()

	clientset, err := s.createClientset(&runtime.Kubeconfig)
	if err != nil {
		retErr := fmt.Errorf("failed to create clientset: %w", err)
		span.RecordError(retErr)
		span.SetStatus(codes.Error, retErr.Error())

		return nil, retErr
	}

	pvcs, err := clientset.CoreV1().PersistentVolumeClaims(corev1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		retErr := fmt.Errorf("failed to list pvcs: %w", err)
		span.RecordError(retErr)
		span.SetStatus(codes.Error, retErr.Error())

		return nil, retErr
	}

	return &Scan{
		pvcs: *pvcs,
	}, nil
}

func (s *Scanner) createClientset(config *rest.Config) (kubernetes.Interface, error) {
	if s.clientFactory == nil {
		return kubernetes.NewForConfig(config)
	}

	return s.clientFactory(config)
}
