package node

import (
	"errors"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/kyma-project/kyma-metrics-collector/pkg/config"
	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
)

const nodeInstanceTypeLabel = "node.kubernetes.io/instance-type"

var ErrUnknownVM = errors.New("unknown provider and node type combination")

var _ resource.ScanConverter = &Scan{}

type Scan struct {
	providerType string
	specs        *config.PublicCloudSpecs

	nodes corev1.NodeList
}

func (s *Scan) UM(duration time.Duration) (resource.UMMeasurement, error) {
	return resource.UMMeasurement{}, nil
}

func (s *Scan) EDP() (resource.EDPMeasurement, error) {
	edp := resource.EDPMeasurement{}

	var errs []error

	vmTypes := make(map[string]int)

	for _, node := range s.nodes.Items {
		nodeType := node.Labels[nodeInstanceTypeLabel]
		nodeType = strings.ToLower(nodeType)

		vmFeature := s.specs.GetFeature(s.providerType, nodeType)
		if vmFeature == nil {
			errs = append(errs, fmt.Errorf("%w: provider: %s, node: %s", ErrUnknownVM, s.providerType, nodeType))
			continue
		}

		edp.ProvisionedCPUs += vmFeature.CpuCores
		edp.ProvisionedRAMGb += vmFeature.Memory
		vmTypes[nodeType] += 1
	}

	for vmType, count := range vmTypes {
		edp.VMTypes = append(edp.VMTypes, resource.VMType{
			Name:  vmType,
			Count: count,
		})
	}

	return edp, errors.Join(errs...)
}
