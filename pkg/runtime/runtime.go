package runtime

import "k8s.io/client-go/rest"

type ProviderType string

const (
	ProviderAzure ProviderType = "azure"
	ProviderAWS   ProviderType = "aws"
	ProviderGCP   ProviderType = "gcp"
	ProviderCCEE  ProviderType = "sapconvergedcloud"
)

type Info struct {
	Kubeconfig   rest.Config
	ProviderType ProviderType
	ShootID      string
}
