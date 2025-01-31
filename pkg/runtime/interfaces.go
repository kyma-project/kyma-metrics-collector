package runtime

import (
	volumesnapshotclientset "github.com/kubernetes-csi/external-snapshotter/client/v6/clientset/versioned"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"
	"k8s.io/client-go/rest"
)

type ConfigProvider interface {
	Get(runtimeID string) ([]byte, error)
}

type ClientFactory interface {
	NewClients(config *rest.Config) (InterfaceCloser, error)
}

type InterfaceCloser interface {
	Interface
	ConnectionCloser
}

type ConnectionCloser interface {
	CloseConnections()
}

type Interface interface {
	Metadata() metadata.Interface
	K8s() kubernetes.Interface
	VolumeSnapshot() volumesnapshotclientset.Interface
	Dynamic() dynamic.Interface
}
