package edp

import "github.com/kyma-project/kyma-metrics-collector/pkg/resource"

type payload struct {
	RuntimeID    string                  `json:"runtime_id"           validate:"required"`
	SubAccountID string                  `json:"sub_account_id"       validate:"required"`
	ShootName    string                  `json:"shoot_name"           validate:"required"`
	Timestamp    string                  `json:"timestamp"            validate:"required"`
	Compute      resource.EDPMeasurement `json:"compute"              validate:"required"`
	Networking   *networking             `json:"networking,omitempty"`
}

type networking struct {
	ProvisionedVnets int `json:"provisioned_vnets" validate:"numeric"`
	ProvisionedIPs   int `json:"provisioned_ips"   validate:"numeric"`
}

func newPayload(runtimeID, subAccountID, shootName, timeStamp string, EDPMeasuremnets []resource.EDPMeasurement) payload {
	aggregatedEDPMeasurement := aggregateEDPMeasurements(EDPMeasuremnets)

	return payload{
		RuntimeID:    runtimeID,
		SubAccountID: subAccountID,
		ShootName:    shootName,
		Timestamp:    timeStamp,
		Compute:      aggregatedEDPMeasurement,
	}
}

func aggregateEDPMeasurements(EDPMeasurements []resource.EDPMeasurement) resource.EDPMeasurement {
	aggregatedEDPMeasurement := resource.EDPMeasurement{}

	for _, m := range EDPMeasurements {
		aggregatedEDPMeasurement.VMTypes = append(aggregatedEDPMeasurement.VMTypes, m.VMTypes...)

		aggregatedEDPMeasurement.ProvisionedCPUs += m.ProvisionedCPUs
		aggregatedEDPMeasurement.ProvisionedRAMGb += m.ProvisionedRAMGb

		aggregatedEDPMeasurement.ProvisionedVolumes.SizeGbTotal += m.ProvisionedVolumes.SizeGbTotal
		aggregatedEDPMeasurement.ProvisionedVolumes.Count += m.ProvisionedVolumes.Count
		aggregatedEDPMeasurement.ProvisionedVolumes.SizeGbRounded += m.ProvisionedVolumes.SizeGbRounded
	}

	return aggregatedEDPMeasurement
}
