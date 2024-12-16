package stubs

import (
	"time"

	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
)

type Scan struct {
	EDPMeasurement resource.EDPMeasurement
	EDPError       error
}

func NewScan(EDPMeasurement resource.EDPMeasurement, EDPError error) Scan {
	return Scan{
		EDPMeasurement: EDPMeasurement,
		EDPError:       EDPError,
	}
}

func (s Scan) UM(duration time.Duration) (resource.UMMeasurement, error) {
	return resource.UMMeasurement{}, nil
}

func (s Scan) EDP() (resource.EDPMeasurement, error) {
	return s.EDPMeasurement, s.EDPError
}
