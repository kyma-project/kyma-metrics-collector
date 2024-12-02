package pvc

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"

	kmccache "github.com/kyma-project/kyma-metrics-collector/pkg/cache"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
	skrcommons "github.com/kyma-project/kyma-metrics-collector/pkg/skr/commons"
)

var fakeShootInfo = kmccache.Record{
	InstanceID:      "adccb200-6052-4192-8adf-785b8a5af306",
	RuntimeID:       "fe5ab5d6-5b0b-4b70-9644-7f89d230b516",
	SubAccountID:    "1ae0dbe1-d13d-4e39-bed4-7c83364084d5",
	GlobalAccountID: "0c22f798-e572-4fc7-a502-cd825c742ff6",
	ShootName:       "c-987654",
}

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

	clientFactory := func(*rest.Config) (kubernetes.Interface, error) {
		clientset := fake.NewSimpleClientset(pvcs)
		return clientset, nil
	}

	scanner := Scanner{
		clientFactory: clientFactory,
	}

	provider := "test-provider"
	result, err := scanner.Scan(context.Background(), &runtime.Info{
		ProviderType: provider,
		ShootInfo:    fakeShootInfo,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	pvcScan, ok := result.(*Scan)
	require.True(t, ok)
	require.Equal(t, pvcs.Items, pvcScan.pvcs.Items)

	gotMetrics, err := skrcommons.TotalQueriesMetric.GetMetricWithLabelValues(
		skrcommons.ListingPVCsAction,
		strconv.FormatBool(true),
		fakeShootInfo.ShootName,
		fakeShootInfo.InstanceID,
		fakeShootInfo.RuntimeID,
		fakeShootInfo.SubAccountID,
		fakeShootInfo.GlobalAccountID,
	)
	require.NoError(t, err)
	require.Equal(t, 1, int(testutil.ToFloat64(gotMetrics)))
}

func TestScanner_Scan_Error(t *testing.T) {
	clientFactory := func(*rest.Config) (kubernetes.Interface, error) {
		clientset := fake.NewSimpleClientset()
		clientset.PrependReactor("list", "persistentvolumeclaims", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
			return true, nil, errors.New("failed to list pvcs")
		})

		return clientset, nil
	}

	scanner := Scanner{
		clientFactory: clientFactory,
	}
	result, err := scanner.Scan(context.Background(), &runtime.Info{ShootInfo: fakeShootInfo})

	require.Error(t, err)
	require.Nil(t, result)

	gotMetrics, err := skrcommons.TotalQueriesMetric.GetMetricWithLabelValues(
		skrcommons.ListingPVCsAction,
		strconv.FormatBool(false),
		fakeShootInfo.ShootName,
		fakeShootInfo.InstanceID,
		fakeShootInfo.RuntimeID,
		fakeShootInfo.SubAccountID,
		fakeShootInfo.GlobalAccountID,
	)
	require.NoError(t, err)
	require.Equal(t, 1, int(testutil.ToFloat64(gotMetrics)))
}
