package vsc

import (
	"errors"
	"math"
	"time"

	v1 "github.com/kubernetes-csi/external-snapshotter/client/v6/apis/volumesnapshot/v1"

	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
)

const (
	// storageRoundingFactor rounds of storage to 32. E.g. 17 -> 32, 33 -> 64.
	storageRoundingFactor = 32

	GiB = 1 << (10 * 3) //nolint:mnd // 1 GiB = 1024^3 bytes
)

var _ resource.ScanConverter = &Scan{}

type Scan struct {
	vscs v1.VolumeSnapshotContentList
}

func (s *Scan) UM(duration time.Duration) (resource.UMMeasurement, error) {
	return resource.UMMeasurement{}, nil
}

func (s *Scan) EDP() (resource.EDPMeasurement, error) {
	edp := resource.EDPMeasurement{}

	var errs []error

	for _, vsc := range s.vscs.Items {
		if vsc.Status.ReadyToUse != nil && *vsc.Status.ReadyToUse {
			currVSC := getSizeInGB(*vsc.Status.RestoreSize)
			edp.ProvisionedVolumes.SizeGbTotal += currVSC
			edp.ProvisionedVolumes.SizeGbRounded += getVolumeRoundedToFactor(currVSC)
			edp.ProvisionedVolumes.Count += 1
		}
	}

	return edp, nil
}

func getSizeInGB(value int64) int64 {
	gVal := int64(float64(value) / GiB)

	return gVal
}

func getVolumeRoundedToFactor(size int64) int64 {
	return int64(math.Ceil(float64(size)/storageRoundingFactor) * storageRoundingFactor)
}
