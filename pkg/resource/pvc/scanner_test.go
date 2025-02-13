package pvc

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime/stubs"
)

func TestScanner_ID(t *testing.T) {
	scanner := Scanner{}
	require.Equal(t, "pvc", string(scanner.ID()), "Scanner ID should be 'pvc'")
}

func TestScanner_Scan_Successful(t *testing.T) {
	pvcs := &corev1.PersistentVolumeClaimList{
		Items: []corev1.PersistentVolumeClaim{
			{ObjectMeta: metav1.ObjectMeta{Name: "pvc1"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "pvc2"}},
		},
	}

	clients := &stubs.Clients{
		KubernetesInterface: fake.NewClientset(pvcs),
	}

	scanner := Scanner{}

	provider := "test-provider"
	result, err := scanner.Scan(t.Context(), &runtime.Info{
		ProviderType: provider,
	}, clients)
	require.NoError(t, err)
	require.NotNil(t, result)

	pvcScan, ok := result.(*Scan)
	require.True(t, ok)
	require.Equal(t, pvcs.Items, pvcScan.pvcs.Items)
}

func TestScanner_Scan_Error(t *testing.T) {
	clientset := fake.NewClientset()
	clientset.PrependReactor("list", "persistentvolumeclaims", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
		return true, nil, errors.New("failed to list pvcs")
	})

	clients := &stubs.Clients{
		KubernetesInterface: clientset,
	}

	scanner := Scanner{}
	result, err := scanner.Scan(t.Context(), &runtime.Info{}, clients)

	require.Error(t, err)
	require.Nil(t, result)
}
