package pvc

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kmcotel "github.com/kyma-project/kyma-metrics-collector/pkg/otel"
	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
)

var _ resource.Scanner = &Scanner{}

type Scanner struct{}

// NewScanner creates a new instance of Scanner.
// While not strictly necessary, this factory function is provided
// for consistency with other scanner implementations.
func NewScanner() *Scanner {
	return &Scanner{}
}

func (s *Scanner) ID() resource.ScannerID {
	return "pvc"
}

func (s *Scanner) Scan(ctx context.Context, runtime *runtime.Info, clients runtime.Interface) (resource.ScanConverter, error) {
	ctx, span := otel.Tracer("").Start(ctx, "pvc_scan", kmcotel.SpanAttributes(runtime))
	defer span.End()

	pvcs, err := clients.K8s().CoreV1().PersistentVolumeClaims(corev1.NamespaceAll).List(ctx, metav1.ListOptions{})
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
