package resource

import (
	"context"
	"time"

	"github.com/kyma-project/kyma-metrics-collector/pkg/process"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
)

type ScannerID string

// Scanner is an interface for measuring a specific resource related to a single cluster.
type Scanner interface {
	// ScanConverter returns the measure for the given clusterid. If an error occurs, the measure is nil.
	// The scan is time dependent and should be taken at the time of the call.
	// The scanner is responsible for exposing metrics about the values retrieved. All measurers should follow a similar pattern.
	// These metrics are just for informational purposes and must not be used for alerting or billing.
	Scan(ctx context.Context, runtime *runtime.Info) (ScanConverter, error)

	// ID returns the ID of the scanner. This name is used to identify the measure in the record.
	ID() ScannerID
}

type UMMeasurementConverter interface {
	// UM returns the measurement data required for creating a unified metering record
	// The duration is the time passed since the last measure was taken.
	UM(duration time.Duration) (UMMeasurement, error)
}

type EDPMeasurementConverter interface {
	// EDP updates the EDPRecord with the measure. All billing logic such as conversion to storage / cpu / memory units must be done here.
	// As the EDPRecord is not time dependent, the duration is not passed.
	EDP(specs *process.PublicCloudSpecs) (EDPMeasurement, error)
}

type ScanConverter interface {
	UMMeasurementConverter
	EDPMeasurementConverter
}
