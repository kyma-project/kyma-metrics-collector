package vsc

import (
	"context"
	"errors"
	"testing"

	v1 "github.com/kubernetes-csi/external-snapshotter/client/v6/apis/volumesnapshot/v1"
	volumesnapshotclientset "github.com/kubernetes-csi/external-snapshotter/client/v6/clientset/versioned"
	"github.com/kubernetes-csi/external-snapshotter/client/v6/clientset/versioned/fake"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"

	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
)

func TestScanner_ID(t *testing.T) {
	scanner := Scanner{}
	require.Equal(t, "vsc", string(scanner.ID()), "Scanner ID should be 'vsc'")
}

func TestScanner_Scan_Successful(t *testing.T) {
	vscs := &v1.VolumeSnapshotContentList{
		Items: []v1.VolumeSnapshotContent{
			{ObjectMeta: metav1.ObjectMeta{Name: "vsc1"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "vsc2"}},
		},
	}

	clientFactory := func(*rest.Config) (volumesnapshotclientset.Interface, error) {
		clientset := fake.NewSimpleClientset(vscs)
		return clientset, nil
	}

	scanner := Scanner{
		clientFactory: clientFactory,
	}

	provider := "test-provider"
	result, err := scanner.Scan(context.Background(), &runtime.Info{
		ProviderType: provider,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	pvcScan, ok := result.(*Scan)
	require.True(t, ok)
	require.Equal(t, vscs.Items, pvcScan.vscs.Items)
}

func TestScanner_Scan_Error(t *testing.T) {
	clientFactory := func(*rest.Config) (volumesnapshotclientset.Interface, error) {
		clientset := fake.NewSimpleClientset()
		clientset.PrependReactor("list", "volumesnapshotcontents", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
			return true, nil, errors.New("failed to list vscs")
		})

		return clientset, nil
	}

	scanner := Scanner{
		clientFactory: clientFactory,
	}
	result, err := scanner.Scan(context.Background(), &runtime.Info{})

	require.Error(t, err)
	require.Nil(t, result)
}
