package pvc

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
	skrcommons "github.com/kyma-project/kyma-metrics-collector/pkg/skr/commons"
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
	ctx, span := otel.Tracer("").Start(ctx, "kmc.pvc_scan")
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
		skrcommons.RecordSKRQuery(false, skrcommons.ListingPVCsAction, runtime.ShootInfo)

		return nil, retErr
	}

	skrcommons.RecordSKRQuery(true, skrcommons.ListingPVCsAction, runtime.ShootInfo)

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
