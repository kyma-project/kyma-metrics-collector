package node

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/metadata/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/kyma-project/kyma-metrics-collector/pkg/config"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime/stubs"
)

func TestScanner_ID(t *testing.T) {
	scanner := Scanner{}
	require.Equal(t, "node", string(scanner.ID()), "Scanner ID should be 'node'")
}

func TestScanner_Scan_Successful(t *testing.T) {
	nodes := &metav1.PartialObjectMetadataList{
		Items: []metav1.PartialObjectMetadata{
			{ObjectMeta: metav1.ObjectMeta{Name: "node1"}, TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Node"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "node2"}, TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Node"}},
		},
	}

	scheme := fake.NewTestScheme()
	scheme.AddKnownTypes(corev1.SchemeGroupVersion, &metav1.PartialObjectMetadata{}, &metav1.PartialObjectMetadataList{})

	clientset := fake.NewSimpleMetadataClient(scheme,
		nodes,
	)

	clients := stubs.Clients{
		MetadataInterface: clientset,
	}

	scanner := Scanner{
		specs: &config.PublicCloudSpecs{},
	}

	provider := "test-provider"

	result, err := scanner.Scan(t.Context(), &runtime.Info{
		ProviderType: provider,
	}, clients)
	require.NoError(t, err)
	require.NotNil(t, result)

	nodeScan, ok := result.(*Scan)
	require.True(t, ok)
	require.Equal(t, provider, nodeScan.providerType)
	require.Equal(t, nodes.Items, nodeScan.list.Items)
	require.Equal(t, scanner.specs, nodeScan.specs)
}

func TestScanner_Scan_Error(t *testing.T) {
	scheme := fake.NewTestScheme()
	scheme.AddKnownTypes(corev1.SchemeGroupVersion, &corev1.Node{}, &corev1.NodeList{})

	clientset := fake.NewSimpleMetadataClient(scheme, &corev1.NodeList{})
	clientset.PrependReactor("list", "nodes", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
		return true, nil, errors.New("failed to list nodes")
	})

	clients := stubs.Clients{
		MetadataInterface: clientset,
	}

	scanner := Scanner{}

	result, err := scanner.Scan(t.Context(), &runtime.Info{}, clients)

	require.Error(t, err)
	require.Nil(t, result)
}
