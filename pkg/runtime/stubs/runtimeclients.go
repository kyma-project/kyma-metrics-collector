package stubs

import (
	volumesnapshotclientset "github.com/kubernetes-csi/external-snapshotter/client/v6/clientset/versioned"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/rest"

	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
)

type RuntimeClients struct {
	MetadataInterface       metadata.Interface
	KubernetesInterface     kubernetes.Interface
	VolumeSnapshotInterface volumesnapshotclientset.Interface
	DynamicInterface        dynamic.Interface
}

func (r RuntimeClients) CloseConnections() {
}

func (r RuntimeClients) Metadata() metadata.Interface {
	return r.MetadataInterface
}

func (r RuntimeClients) K8s() kubernetes.Interface {
	return r.KubernetesInterface
}

func (r RuntimeClients) VolumeSnapshot() volumesnapshotclientset.Interface {
	return r.VolumeSnapshotInterface
}

func (r RuntimeClients) Dynamic() dynamic.Interface {
	return r.DynamicInterface
}

type RuntimeClientFactory struct {
	RuntimeClients
}

func (r RuntimeClientFactory) NewClients(config *rest.Config) (runtime.InterfaceCloser, error) {
	return r.RuntimeClients, nil
}
