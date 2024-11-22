package measurer

import (
	"context"
	"time"

	"k8s.io/client-go/rest"

	"github.com/kyma-project/kyma-metrics-collector/pkg/process"
)

type MeasurerID string

// Measurer is an interface for measuring a specific resource related to a single cluster
type Measurer interface {
	// Measure returns the measure for the given clusterid. If an error occurs, the measure is nil.
	// The measure is time dependent and should be taken at the time of the call.
	// The measurer is responsible for exposing metrics about the values retrieved. All measurers should follow a similar pattern.
	// These metrics are just for informational purposes and must not be used for alerting or billing.
	Measure(ctx context.Context, config *rest.Config) (Measurement, error)

	// ID returns the ID of the measurer. This name is used to identify the measure in the record.
	ID() MeasurerID
}

type Measurement interface {
	// UpdateUM updates the UMRecord with the measure. All billing logic such as convertion to capacity units must be done here.
	// The duration is the time passed since the last measure was taken.
	UM(duration time.Duration) (UMData, error)

	// UpdateEDP updates the EDPRecord with the measure. All billing logic such as convertion to storage / cpu / memory units must be done here.
	// As the EDPRecord is not time dependent, the duration is not passed.
	EDP(specs *process.PublicCloudSpecs) (EDPData, error)
}
