package kubeconfigprovider

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/jellydator/ttlcache/v3"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

var ErrNotFound = errors.New("Item not found in cache")

type KubeconfigProvider struct {
	cache  *ttlcache.Cache[string, []byte]
	client v1.CoreV1Interface
	ttl    time.Duration
	logger *zap.SugaredLogger
}

func New(client v1.CoreV1Interface, logger *zap.SugaredLogger, ttl time.Duration) *KubeconfigProvider {
	loader := ttlcache.LoaderFunc[string, []byte](
		func(c *ttlcache.Cache[string, []byte], key string) *ttlcache.Item[string, []byte] {
			kubeconfig, err := getKubeConfigFromSecret(logger, client, key)
			if err != nil {
				logger.Errorf("kubeconfig kubeconfigprovider failed to get kubeconfig for cluster (runtimeID: %s) from secret: %s",
					key, err)
				return nil
			}

			return c.Set(key, kubeconfig, getJitterTTL(ttl))
		},
	)

	return &KubeconfigProvider{
		client: client,
		cache: ttlcache.New[string, []byte](
			ttlcache.WithTTL[string, []byte](ttl),
			ttlcache.WithDisableTouchOnHit[string, []byte](),
			ttlcache.WithLoader[string, []byte](loader),
		),
		ttl:    ttl,
		logger: logger,
	}
}

// GetKubeConfigFromCache returns the kubeconfig from the kubeconfigprovider if it is not expired.
// If it is expired, it will get the kubeconfig from the secret and set it in the kubeconfigprovider.
func (k *KubeconfigProvider) Get(runtimeID string) ([]byte,
	error,
) {
	k.cache.DeleteExpired()
	recordMetrics()

	item := k.cache.Get(runtimeID)
	if item == nil {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, runtimeID)
	}

	return item.Value(), nil
}

// getkubeConfigFromSecret gets the kubeconfig from the secret.
func getKubeConfigFromSecret(logger *zap.SugaredLogger, client v1.CoreV1Interface, runtimeID string) ([]byte,
	error,
) {
	secretResourceName := fmt.Sprintf("kubeconfig-%s", runtimeID)

	secret, err := getKubeConfigSecret(logger, client, runtimeID, secretResourceName)
	if err != nil {
		return nil, err
	}

	kubeconfig, found := secret.Data["config"]
	if !found {
		return nil, fmt.Errorf("kubeconfig kubeconfigprovider found kubeconfig-secret '%s' for runtime '%s' which does not include the data-key 'config'",
			secretResourceName, runtimeID)
	}

	if len(kubeconfig) == 0 {
		return nil, fmt.Errorf("kubeconfig kubeconfigprovider found kubeconfig-secret '%s' for runtime '%s' which includes an empty kubeconfig string",
			secretResourceName, runtimeID)
	}

	return kubeconfig, nil
}

// getKubeConfigSecret gets the kubeconfig secret from the cluster.
func getKubeConfigSecret(logger *zap.SugaredLogger,
	client v1.CoreV1Interface,
	runtimeID, secretResourceName string,
) (*corev1.Secret, error) {
	secret, err := client.Secrets("kcp-system").Get(context.Background(), secretResourceName, metav1.GetOptions{})
	if err != nil {
		if k8serr.IsNotFound(err) { // accepted failure
			logger.Debugf("kubeconfig kubeconfigprovider cannot find a kubeconfig-secret '%s' for cluster with runtimeID %s: %s",
				secretResourceName, runtimeID, err)
			return nil, err
		} else if k8serr.IsForbidden(err) { // configuration failure
			logger.Errorf("kubeconfig kubeconfigprovider is not allowed to lookup kubeconfig-secret '%s' for cluster with runtimeID %s: %s",
				secretResourceName, runtimeID, err)
			return nil, err
		}

		logger.Errorf("kubeconfig kubeconfigprovider failed to lookup kubeconfig-secret '%s' for cluster with runtimeID %s: %s",
			secretResourceName, runtimeID, err)

		return nil, err
	}

	return secret, nil
}

func getJitterTTL(ttl time.Duration) time.Duration {
	maxTTL := ttl
	buffer := int64(maxTTL.Minutes() / 3) //nolint:mnd // we accept TTLS with 1/3 length above maxTTL
	jitter := rand.Int63n(buffer) + int64(maxTTL.Minutes())

	return time.Duration(jitter) * time.Minute
}
