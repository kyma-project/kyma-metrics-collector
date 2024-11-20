# 1. Collector Structure

Date: 2024-11-20

## Status

Proposed

## Context

The current implementation of KMCs SKR processing has been built with a monolithic architecture. This has led to a number of issues including:
- a single issue processing an SKR renders the entire billing process for this cluster invalid
- adding new ressources to the billing process is difficult
- the current implementation is hard to test
- every client needs to implement 

## Decision

We will refactor the current implementation of the SKR processing to a more modular architecture. This will allow us to:
- process each measure independently
- add new measures easily
- each measure will contain its own logic for processing the measure. Processing the measure will include:
  - updating the UM record (converting the measure to capacity units, adding functional metrics)
  - updating the EDP record (converting the measure to storage / cpu / memory units)

The interfaces and their purpose is defined as follows:
- `Measurer` is an interface for measuring a specific resource related to a single cluster
- `Measure` is an interface for processing a measure
- `CollectorSender` is an interface for collecting and sending the measures to the backend

Collectors call the measurers to get the measures for a cluster. The collector then processes the measures and sends them to the backend.
All processed measures are stored in a map with the name of the measurer as the key. This map will then be stored for the next run of the collector.

```go
package measuring

import (
	"time"

	"go.uber.org/zap"
)

type UMRecord struct {
	CU                int
	FunctionalMetrics map[string]int
}

type EDPRecord struct {
	Storage int
	CPU     int
	Memory  int
}

// Measurer is an interface for measuring a specific resource related to a single cluster
type Measurer interface {
	// Measure returns the measure for the given clusterid. If an error occurs, the measure is nil.
	// The measure is time dependent and should be taken at the time of the call.
	// The measurer is responsible for exposing metrics about the values retrieved. All measurers should follow a similar pattern. 
	// These metrics are just for informational purposes and must not be used for alerting or billing.
	Measure(clusterid string) (Measure, error)
	
    // Name returns the name of the measurer. This name is used to identify the measure in the record.
	Name() string
}

type Measure interface {
	// UpdateUM updates the UMRecord with the measure. All billing logic such as convertion to capacity units must be done here.
	// The duration is the time passed since the last measure was taken.
	UpdateUM(record *UMRecord, duration time.Duration)

	// UpdateEDP updates the EDPRecord with the measure. All billing logic such as convertion to storage / cpu / memory units must be done here.
	// As the EDPRecord is not time dependent, the duration is not passed.
	UpdateEDP(record *EDPRecord)
}

type RedisMeasure struct {
	instances map[string]int
}

func (rm *RedisMeasure) UpdateUM(record *UMRecord, duration time.Duration) {
	for tier, instance := range rm.instances {
		record.CU += calculateCU(tier, instance)
	}
	record.FunctionalMetrics["redis"] += 1
}

// calculateCU calculates the capacity units for a given tier and instance
func calculateCU(tier string, instance int) int {
	return 1
}

func (rm *RedisMeasure) UpdateEDP(record *EDPRecord) {
	record.Storage = 1
	record.CPU = 2
	record.Memory = 3

}

var _ Measure = &RedisMeasure{}

func NewUMRecord(from, to time.Time, measures map[any]Measure) *UMRecord {
	record := &UMRecord{}
	dur := to.Sub(from)
	for _, measure := range measures {
		measure.UpdateUM(record, dur)
	}
	return record
}

type CollectorSender interface {
	// CollectAndSend collects and sends the measures to the backend. It returns the measures collected.
	CollectAndSend(clusterid string, previousMeasures map[any]Measure) (map[any]Measure, error)
}

type UMCollector struct {
	measurer []Measurer
	logger   *zap.Logger
}

var _ CollectorSender = &UMCollector{}

func NewCollector(measurer ...Measurer) CollectorSender {
	return &UMCollector{
		measurer: measurer,
	}
}

func (c *UMCollector) CollectAndSend(clusterid string, previousMeasures map[any]Measure) (map[any]Measure, error) {
	measures := make(map[any]Measure)
	for _, m := range c.measurer {

		// record metrics about success/failure
		// record spans for timing
		measure, err := m.Measure(clusterid)
		if err != nil {
			// log errors here, but continue with other measures
			c.logger.Error("error measuring", zap.Error(err))
			// use previous measure
			measure = previousMeasures[m.Name()]
		}
		// use new or old measure
		measures[m.Name()] = measure
	}

	record := NewUMRecord(time.Now(), time.Now(), measures)
	err := c.sendRecord(record)
	return measures, err
	// use new or old measure
}

// sendRecord sends the record to the UM backend
func (c *UMCollector) sendRecord(record *UMRecord) error {
	return nil
}

type RedisMeasurer struct {
}

func (r RedisMeasurer) Measure(clusterid string) (Measure, error) {
	// Collect redis data from the cluster
	// return the measure
	return &RedisMeasure{}, nil
}

func (r RedisMeasurer) Name() string {
	return "redis"
}

var _ Measurer = &RedisMeasurer{}

```
