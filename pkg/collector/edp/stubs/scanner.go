package stubs

import (
	"context"

	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
)

type Scanner struct {
	scanConverter resource.ScanConverter
	scanError     error
	scannerID     resource.ScannerID
}

func NewScanner(scanConverter resource.ScanConverter, scanError error, ID resource.ScannerID) Scanner {
	return Scanner{
		scanConverter: scanConverter,
		scanError:     scanError,
		scannerID:     ID,
	}
}

func (s Scanner) Scan(ctx context.Context, runtime *runtime.Info, clients runtime.Interface) (resource.ScanConverter, error) {
	return s.scanConverter, s.scanError
}

func (s Scanner) ID() resource.ScannerID {
	return s.scannerID
}
