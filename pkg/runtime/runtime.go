package runtime

import (
	"k8s.io/client-go/rest"
	"net/http"
)

type Info struct {
	InstanceID      string
	RuntimeID       string
	SubAccountID    string
	GlobalAccountID string
	ShootName       string
	ProviderType    string
	Kubeconfig      rest.Config
	Client          *http.Client
}
