package vsc

import (
	"context"
	"fmt"

	volumesnapshotclientset "github.com/kubernetes-csi/external-snapshotter/client/v6/clientset/versioned"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
)

var _ resource.Scanner = &Scanner{}

type Scanner struct {
	clientFactory func(config *rest.Config) (volumesnapshotclientset.Interface, error)
}

func NewScanner() *Scanner {
	return &Scanner{}
}

func (s *Scanner) ID() resource.ScannerID {
	return "vsc"
}

func (s *Scanner) Scan(ctx context.Context, runtime *runtime.Info) (resource.ScanConverter, error) {
	ctx, span := otel.Tracer("").Start(ctx, "kmc.vsc_scan",
		trace.WithAttributes(
			attribute.String("instance_id", runtime.InstanceID),
			attribute.String("runtime_id", runtime.RuntimeID),
			attribute.String("sub_account_id", runtime.SubAccountID),
			attribute.String("global_account_id", runtime.GlobalAccountID),
			attribute.String("shoot_name", runtime.ShootName),
			attribute.String("provider", runtime.ProviderType),
		),
	)
	defer span.End()

	clientset, err := s.createClientSet(&runtime.Kubeconfig)
	if err != nil {
		retErr := fmt.Errorf("failed to create clientset: %w", err)
		span.RecordError(retErr)
		span.SetStatus(codes.Error, retErr.Error())

		return nil, retErr
	}

	vscs, err := clientset.SnapshotV1().VolumeSnapshotContents().List(ctx, metav1.ListOptions{})
	if err != nil {
		retErr := fmt.Errorf("failed to list vscs: %w", err)
		span.RecordError(retErr)
		span.SetStatus(codes.Error, retErr.Error())

		return nil, retErr
	}

	return &Scan{
		vscs: *vscs,
	}, nil
}

func (s *Scanner) createClientSet(config *rest.Config) (volumesnapshotclientset.Interface, error) {
	if s.clientFactory == nil {
		return volumesnapshotclientset.NewForConfig(config)
	}

	return s.clientFactory(config)
}
