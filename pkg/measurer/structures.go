package measurer

type VMType struct {
	Name  string `json:"name"  validate:"required"`
	Count int    `json:"count" validate:"numeric"`
}

type EDPData struct {
	VMTypes            []VMType           `json:"vm_types"            validate:"required"`
	ProvisionedCpus    int                `json:"provisioned_cpus"    validate:"numeric"`
	ProvisionedRAMGb   float64            `json:"provisioned_ram_gb"  validate:"numeric"`
	ProvisionedVolumes ProvisionedVolumes `json:"provisioned_volumes" validate:"required"`
}

type ProvisionedVolumes struct {
	SizeGbTotal   int64 `json:"size_gb_total"   validate:"numeric"`
	Count         int   `json:"count"           validate:"numeric"`
	SizeGbRounded int64 `json:"size_gb_rounded" validate:"numeric"`
}

type UMData struct {
}
