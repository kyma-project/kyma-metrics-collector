package runtime

import "k8s.io/client-go/rest"

type Info struct {
	Kubeconfig   rest.Config
	ProviderType string
	ShootID      string
}
