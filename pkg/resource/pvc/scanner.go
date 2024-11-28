package pvc

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
)

const (
	tracerName = "kmc.resource.pvc"
)

var _ resource.Scanner = &Scanner{}

type Scanner struct{}

func (s Scanner) ID() resource.ScannerID {
	return "pvc"
}

func (s Scanner) Scan(ctx context.Context, runtime *runtime.Info) (resource.ScanConverter, error) {
	ctx, span := otel.Tracer(tracerName).Start(ctx, "Scanner.Scan")
	defer span.End()

	clientset, err := kubernetes.NewForConfig(&runtime.Kubeconfig)
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
