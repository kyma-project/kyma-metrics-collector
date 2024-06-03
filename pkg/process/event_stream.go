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

	Azure = "azure"
	AWS   = "aws"
	GCP   = "gcp"
	CCEE  = "sapconvergedcloud"
)

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

	nfsPVCStorage := int64(0)
	nfsPVCStorageRounded := int64(0)
	nfsVolumeCount := 0

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
			if pvc.GetObjectMeta().GetAnnotations()["volume.beta.kubernetes.io/storage-class"] == "nfs" {
				if pvc.Status.Phase == corev1.ClaimBound {
					currPVC := getSizeInGB(pvc.Status.Capacity.Storage())
					nfsPVCStorage += currPVC
					nfsPVCStorageRounded += getVolumeRoundedToFactor(currPVC)
					nfsVolumeCount += 1
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

	metric.Compute.ProvisionedNFSVolumes.SizeGbTotal = nfsPVCStorage
	metric.Compute.ProvisionedNFSVolumes.SizeGbRounded = nfsPVCStorageRounded
	metric.Compute.ProvisionedNFSVolumes.Count = nfsVolumeCount

	for vmType, count := range vmTypes {
		metric.Compute.VMTypes = append(metric.Compute.VMTypes, edp.VMType{
			Name:  vmType,
			Count: count,
		})
	}

	return metric, nil
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
	gbUnit := math.Pow(2, 30)

	// Converting back from milli to original
	gVal := int64((float64(milliVal) / float64(gbUnit)) / 1000)
	return gVal
}
