package collector

import (
	"github.com/kyma-project/kyma-metrics-collector/pkg/measurement"

	"github.com/kyma-project/kyma-metrics-collector/pkg/measurer"
)

type MeasurementMap map[measurer.MeasurerID]measurement.Measurement
type CollectorSender interface {
	// CollectAndSend collects and sends the measures to the backend. It returns the measures collected.
	CollectAndSend(clusterid string, previousMeasures MeasurementMap) (MeasurementMap, error)
}
