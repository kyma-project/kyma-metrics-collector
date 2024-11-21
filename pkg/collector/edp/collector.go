package unifiedmetering

import (
	"maps"
	"time"

	"go.uber.org/zap"

	"github.com/kyma-project/kyma-metrics-collector/pkg/collector"
	"github.com/kyma-project/kyma-metrics-collector/pkg/measurer"
)

type Collector struct {
	measurers []measurer.Measurer
	logger    *zap.Logger
}

var _ collector.CollectorSender = &Collector{}

func NewCollector(measurer ...measurer.Measurer) collector.CollectorSender {
	return &Collector{
		measurers: measurer,
	}
}

func (c *Collector) CollectAndSend(clusterid string, previousMeasures collector.MeasurementMap) (collector.MeasurementMap, error) {
	measures := make(collector.MeasurementMap)
	for _, m := range c.measurers {

		// record metrics about success/failure
		// record spans for timing
		msr, err := m.Measure(clusterid)
		if err != nil {
			// log errors here, but continue with other measures
			c.logger.Error("error measuring", zap.Error(err))
			// use previous measure
			msr = previousMeasures[m.ID()]
		}
		// use new or old measure
		measures[m.ID()] = msr
	}

	record := NewRecord(time.Now(), time.Now(), maps.Values(measures))
	err := c.sendRecord(record)
	return measures, err
	// use new or old measure
}

// sendRecord sends the record to the UM backend
func (c *Collector) sendRecord(record *Record) error {
	return nil
}
