package kubeconfigprovider

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestKubeconfigProvider_Get(t *testing.T) {
	cs := fake.NewClientset()
	callCount := 0

	cs.PrependReactor("get", "secrets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		callCount++

		return true, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "kubeconfig-test",
			},
			Data: map[string][]byte{
				"config": []byte("test"),
			},
		}, nil
	})

	looger := zap.NewExample().Sugar()
	provider := New(cs.CoreV1(), looger, 1*time.Second, "test")
	require.Equal(t, 0, provider.cache.Len())

	// Get the kubeconfig from the kubeconfigprovider. Expect the cache to be missed.
	got, err := provider.Get("test")
	require.NoError(t, err)
	require.Equal(t, 1, callCount)
	require.Equal(t, 1, provider.cache.Len())
	require.Equal(t, []byte("test"), got)

	// Get the kubeconfig from the kubeconfigprovider again. Expect the cache to be hit.
	got, err = provider.Get("test")
	require.Equal(t, 1, provider.cache.Len())
	require.NoError(t, err)
	require.Equal(t, []byte("test"), got)
	require.Equal(t, 1, callCount)

	// Wait for the cache to expire
	time.Sleep(1 * time.Second)
	provider.cache.DeleteExpired()
	require.Equal(t, 0, provider.cache.Len())

	// Get the kubeconfig from the kubeconfigprovider again. Expect the cache to be missed.
	got, err = provider.Get("test")
	require.Equal(t, 1, provider.cache.Len())
	require.NoError(t, err)
	require.Equal(t, []byte("test"), got)
	require.Equal(t, 2, callCount)
}
