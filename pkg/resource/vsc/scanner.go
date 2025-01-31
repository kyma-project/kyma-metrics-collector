package vsc

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kmcotel "github.com/kyma-project/kyma-metrics-collector/pkg/otel"
	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
)

var _ resource.Scanner = &Scanner{}

type Scanner struct{}

func NewScanner() *Scanner {
	return &Scanner{}
}

func (s *Scanner) ID() resource.ScannerID {
	return "vsc"
}

func (s *Scanner) Scan(ctx context.Context, runtime *runtime.Info, clients runtime.Interface) (resource.ScanConverter, error) {
	ctx, span := otel.Tracer("").Start(ctx, "vsc_scan", kmcotel.SpanAttributes(runtime))
	defer span.End()

	vscs, err := clients.VolumeSnapshot().SnapshotV1().VolumeSnapshotContents().List(ctx, metav1.ListOptions{})
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
