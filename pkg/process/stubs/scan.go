package stubs

import (
	"time"

	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
)

type Scan struct {
	resources []string
}

func NewScan(resources []string) Scan {
	return Scan{
		resources: resources,
	}
}

func (s Scan) UM(duration time.Duration) (resource.UMMeasurement, error) {
	return resource.UMMeasurement{}, nil
}

func (s Scan) EDP() (resource.EDPMeasurement, error) {
	return resource.EDPMeasurement{}, nil
}
