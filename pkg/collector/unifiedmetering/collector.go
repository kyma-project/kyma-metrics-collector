package unifiedmetering

import (
	"context"
	"maps"
	"time"

	"go.uber.org/zap"

	"github.com/kyma-project/kyma-metrics-collector/pkg/collector"
	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
)

type Collector struct {
	scanners []resource.Scanner
	logger   *zap.Logger
}

var _ collector.CollectorSender = &Collector{}

func NewCollector(scanner ...resource.Scanner) collector.CollectorSender {
	return &Collector{
		scanners: scanner,
	}
}

func (c *Collector) CollectAndSend(ctx context.Context, runtime *runtime.Info, previousScans collector.ScanMap) (collector.ScanMap, error) {
	measures := make(collector.ScanMap)

	for _, s := range c.scanners {
		// record metrics about success/failure
		// record spans for timing
		scan, err := s.Scan(ctx, runtime)
		if err != nil {
			// log errors here, but continue with other measures
			c.logger.Error("error measuring", zap.Error(err))
			// use previous measure
			scan = previousScans[s.ID()]
		}
		// use new or old measure
		measures[s.ID()] = scan
	}

	record := NewRecord(time.Now(), time.Now(), maps.Values(measures))
	err := c.sendRecord(record)

	return measures, err
	// use new or old measure
}

// sendRecord sends the record to the UM backend.
func (c *Collector) sendRecord(record *Record) error {
	return nil
}
