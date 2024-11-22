package edp

import (
	"iter"
	"time"

	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
)

type Record struct {
}

func NewRecord(from, to time.Time, scans iter.Seq[resource.Scan]) *Record {
	return &Record{}
}
