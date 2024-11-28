package service

import (
	"errors"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/kyma-project/kyma-metrics-collector/pkg/config"
	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
)

var _ resource.ScanConverter = &Scan{}

type Scan struct {
	services corev1.ServiceList
}

func (s *Scan) UM(duration time.Duration) (resource.UMMeasurement, error) {
	return resource.UMMeasurement{}, nil
}

func (s *Scan) EDP(specs *config.PublicCloudSpecs) (resource.EDPMeasurement, error) {
	edp := resource.EDPMeasurement{}
	var errs []error

	return edp, errors.Join(errs...)
}
