package process

import (
	"fmt"
	"math"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/kyma-project/kyma-metrics-collector/pkg/edp"
)

const (
	nodeInstanceTypeLabel = "node.kubernetes.io/instance-type"
	// storageRoundingFactor rounds of storage to 32. E.g. 17 -> 32, 33 -> 64.
	storageRoundingFactor = 32

	// nfsPriceMultiplier is the factor by which the NFS PVCs are multiplied to compensate for the higher price.
	nfsPriceMultiplier = 3

	Azure = "azure"
	AWS   = "aws"
	GCP   = "gcp"
	CCEE  = "sapconvergedcloud"
)

const (
	GiB = 1 << (10 * 3) //nolint:mnd // 1 GiB = 1024^3 bytes
)

var nfsLabels = map[string]string{
	"app.kubernetes.io/component":  "cloud-manager",
	"app.kubernetes.io/part-of":    "kyma",
	"app.kubernetes.io/managed-by": "cloud-manager",
}

type EventStream struct {
	KubeConfig string
	Metric     edp.ConsumptionMetrics
}

type Input struct {
	provider string
	nodeList *corev1.NodeList
	pvcList  *corev1.PersistentVolumeClaimList
	svcList  *corev1.ServiceList
}

func (inp Input) Parse(providers *Providers) (*edp.ConsumptionMetrics, error) {
	if inp.nodeList == nil {
		return nil, fmt.Errorf("no nodes data to compute metrics on")
	}

	metric := new(edp.ConsumptionMetrics)
	provisionedCPUs := 0
	provisionedMemory := 0.0
	providerType := inp.provider
	vmTypes := make(map[string]int)

	pvcStorage := int64(0)
	pvcStorageRounded := int64(0)
	volumeCount := 0

	for _, node := range inp.nodeList.Items {
		nodeType := node.Labels[nodeInstanceTypeLabel]
		nodeType = strings.ToLower(nodeType)

		// Calculate CPU and Memory
		vmFeature := providers.GetFeature(providerType, nodeType)
		if vmFeature == nil {
			return nil, fmt.Errorf("providerType: %s and nodeType: %s does not exist in the map", providerType, nodeType)
		}

		provisionedCPUs += vmFeature.CpuCores
		provisionedMemory += vmFeature.Memory
		vmTypes[nodeType] += 1
	}

	if inp.pvcList != nil {
		// Calculate storage from PVCs
		for _, pvc := range inp.pvcList.Items {
			// if the pvc has all labels defined in nfsLabels then it is an NFS PVC
			if hasAllLabels(pvc.Labels, nfsLabels) {
				if pvc.Status.Phase == corev1.ClaimBound {
					currPVC := getSizeInGB(pvc.Status.Capacity.Storage())
					// for NFS PVCs we multiply the used capacity by 3 to compensate for the higher price
					nfsPVCStorage := currPVC * nfsPriceMultiplier
					pvcStorage += nfsPVCStorage
					pvcStorageRounded += getVolumeRoundedToFactor(nfsPVCStorage)
					volumeCount += 1
				}

				continue
			}

			if pvc.Status.Phase == corev1.ClaimBound {
				currPVC := getSizeInGB(pvc.Status.Capacity.Storage())
				pvcStorage += currPVC
				pvcStorageRounded += getVolumeRoundedToFactor(currPVC)
				volumeCount += 1
			}
		}
	}

	// Calculate vnets(for Azure) or vpc(for AWS)
	metric.Timestamp = getTimestampNow()
	metric.Compute.ProvisionedCpus = provisionedCPUs
	metric.Compute.ProvisionedRAMGb = provisionedMemory

	metric.Compute.ProvisionedVolumes.SizeGbTotal = pvcStorage
	metric.Compute.ProvisionedVolumes.SizeGbRounded = pvcStorageRounded
	metric.Compute.ProvisionedVolumes.Count = volumeCount

	for vmType, count := range vmTypes {
		metric.Compute.VMTypes = append(metric.Compute.VMTypes, edp.VMType{
			Name:  vmType,
			Count: count,
		})
	}

	return metric, nil
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

// getTimestampNow returns the time now in the format of RFC3339.
func getTimestampNow() string {
	return time.Now().Format(time.RFC3339)
}

func getVolumeRoundedToFactor(size int64) int64 {
	return int64(math.Ceil(float64(size)/storageRoundingFactor) * storageRoundingFactor)
}

// getSizeInGB converts any value in binarySI representation to GB
// More info: https://github.com/kubernetes/apimachinery/blob/master/pkg/api/resource/quantity.go#L31
func getSizeInGB(value *resource.Quantity) int64 {
	// Converting to milli to normalize
	milliVal := value.MilliValue()

	// Converting back from milli to original
	gVal := int64((float64(milliVal) / GiB) / 1000) //nolint:mnd // 1000 is the factor to convert from milli to original

	return gVal
}
