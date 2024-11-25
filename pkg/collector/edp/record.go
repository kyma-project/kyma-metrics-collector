package edp

import (
	"iter"
	"time"

	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
)

type Record struct {
}

func NewRecord(from, to time.Time, scans iter.Seq[resource.ScanConverter]) *Record {
	for i := range scans {

	}
	return &Record{}
}
