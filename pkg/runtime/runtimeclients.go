package runtime

import (
	"net/http"

	volumesnapshotclientset "github.com/kubernetes-csi/external-snapshotter/client/v6/clientset/versioned"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/rest"
)

type Clients struct {
	metadata       metadata.Interface
	core           kubernetes.Interface
	volumeSnapshot volumesnapshotclientset.Interface
	dynamic        dynamic.Interface
	client         *http.Client
}

type RuntimeClientFactory struct{}

func (r RuntimeClientFactory) NewClients(config *rest.Config) (Interface, error) {
	return NewClients(config)
}

func NewClients(config *rest.Config) (*Clients, error) {
	config.Proxy = http.ProxyFromEnvironment

	cl, err := rest.HTTPClientFor(config)
	if err != nil {
		return nil, err
	}

	core, err := kubernetes.NewForConfigAndClient(config, cl)
	if err != nil {
		return nil, err
	}

	meta, err := metadata.NewForConfigAndClient(config, cl)
	if err != nil {
		return nil, err
	}

	dyn, err := dynamic.NewForConfigAndClient(config, cl)
	if err != nil {
		return nil, err
	}

	volumeSnapshot, err := volumesnapshotclientset.NewForConfigAndClient(config, cl)
	if err != nil {
		return nil, err
	}

	clients := &Clients{
		metadata:       meta,
		core:           core,
		volumeSnapshot: volumeSnapshot,
		dynamic:        dyn,
		client:         cl,
	}

	return clients, nil
}

func (r *Clients) Metadata() metadata.Interface {
	return r.metadata
}

func (r *Clients) K8s() kubernetes.Interface {
	return r.core
}

func (r *Clients) VolumeSnapshot() volumesnapshotclientset.Interface {
	return r.volumeSnapshot
}

func (r *Clients) Dynamic() dynamic.Interface {
	return r.dynamic
}

func (r *Clients) CloseConnections() {
	r.client.CloseIdleConnections()
}
