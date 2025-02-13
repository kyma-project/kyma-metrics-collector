package stubs

import (
	volumesnapshotclientset "github.com/kubernetes-csi/external-snapshotter/client/v8/clientset/versioned"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/rest"

	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
)

type Clients struct {
	MetadataInterface       metadata.Interface
	KubernetesInterface     kubernetes.Interface
	VolumeSnapshotInterface volumesnapshotclientset.Interface
	DynamicInterface        dynamic.Interface
}

func (r Clients) CloseConnections() {
}

func (r Clients) Metadata() metadata.Interface {
	return r.MetadataInterface
}

func (r Clients) K8s() kubernetes.Interface {
	return r.KubernetesInterface
}

func (r Clients) VolumeSnapshot() volumesnapshotclientset.Interface {
	return r.VolumeSnapshotInterface
}

func (r Clients) Dynamic() dynamic.Interface {
	return r.DynamicInterface
}

type ClientFactory struct {
	Clients
}

func (r ClientFactory) NewClients(config *rest.Config) (runtime.InterfaceCloser, error) {
	return r.Clients, nil
}
