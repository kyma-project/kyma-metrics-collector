package pvc

import (
	"errors"
	"math"
	"time"

	corev1 "k8s.io/api/core/v1"
	apiresource "k8s.io/apimachinery/pkg/api/resource"

	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
)

const (
	// storageRoundingFactor rounds of storage to 32. E.g. 17 -> 32, 33 -> 64.
	storageRoundingFactor = 32

	// nfsPriceMultiplier is the factor by which the NFS PVCs are multiplied to compensate for the higher price.
	nfsPriceMultiplier = 3

	GiB = 1 << (10 * 3) //nolint:mnd // 1 GiB = 1024^3 bytes
)

var nfsLabels = map[string]string{
	"app.kubernetes.io/component":  "cloud-manager",
	"app.kubernetes.io/part-of":    "kyma",
	"app.kubernetes.io/managed-by": "cloud-manager",
}

const nfsCapacityLabel = "cloud-resources.kyma-project.io/nfsVolumeStorageCapacity"

var _ resource.ScanConverter = &Scan{}

type Scan struct {
	pvcs corev1.PersistentVolumeClaimList
}

func (s *Scan) UM(duration time.Duration) (resource.UMMeasurement, error) {
	return resource.UMMeasurement{}, nil
}

func (s *Scan) EDP() (resource.EDPMeasurement, error) {
	edp := resource.EDPMeasurement{}

	var errs []error

	for _, pvc := range s.pvcs.Items {
		// if the pvc has all labels defined in nfsLabels then it is an NFS PVC
		if hasAllLabels(pvc.Labels, nfsLabels) {
			if pvc.Status.Phase == corev1.ClaimBound {
				currPVC := getSizeInGB(pvc.Status.Capacity.Storage())

				// label is used as primary source of truth for size - when present, and valid
				if sizeFromLabel, ok := pvc.Labels[nfsCapacityLabel]; ok {
					quantityFromLabel, err := apiresource.ParseQuantity(sizeFromLabel)
					if err == nil {
						currPVC = getSizeInGB(&quantityFromLabel)
					}
				}

				// for NFS PVCs we multiply the used capacity by 3 to compensate for the higher price
				nfsPVCStorage := currPVC * nfsPriceMultiplier
				edp.ProvisionedVolumes.SizeGbTotal += nfsPVCStorage
				edp.ProvisionedVolumes.SizeGbRounded += getVolumeRoundedToFactor(nfsPVCStorage)
				edp.ProvisionedVolumes.Count += 1
			}

			continue
		}

		if pvc.Status.Phase == corev1.ClaimBound {
			currPVC := getSizeInGB(pvc.Status.Capacity.Storage())
			edp.ProvisionedVolumes.SizeGbTotal += currPVC
			edp.ProvisionedVolumes.SizeGbRounded += getVolumeRoundedToFactor(currPVC)
			edp.ProvisionedVolumes.Count += 1
		}
	}

	return edp, errors.Join(errs...)
}

// hasAllLabels checks if the labels map contains all the labels in want.
func hasAllLabels(has, want map[string]string) bool {
	for k, v := range want {
		if has[k] != v {
			return false
		}
	}

	return true
}

func getVolumeRoundedToFactor(size int64) int64 {
	return int64(math.Ceil(float64(size)/storageRoundingFactor) * storageRoundingFactor)
}

// getSizeInGB converts any value in binarySI representation to GB
// More info: https://github.com/kubernetes/apimachinery/blob/master/pkg/api/resource/quantity.go#L31
func getSizeInGB(value *apiresource.Quantity) int64 {
	// Converting to milli to normalize
	milliVal := value.MilliValue()

	// Converting back from milli to original
	gVal := int64((float64(milliVal) / GiB) / 1000) //nolint:mnd // 1000 is the factor to convert from milli to original

	return gVal
}
