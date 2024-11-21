package unifiedmetering

import (
	"iter"
	"time"

	"github.com/kyma-project/kyma-metrics-collector/pkg/measurement"
)

type Record struct {
}

func NewRecord(from, to time.Time, measurements iter.Seq[measurement.Measurement]) *Record {
	return &Record{}
}
