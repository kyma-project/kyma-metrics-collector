package runtime

import "k8s.io/client-go/rest"

type ProviderType string

type Info struct {
	Kubeconfig   rest.Config
	ProviderType ProviderType
}
