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

// ErrNotFound is returned when an item is not found in the cache.
var ErrNotFound = errors.New("Item not found in cache")

const (
	// namespace where the kubeconfig secrets are stored.
	kcpNamespace = "kcp-system"
	// prefix for the kubeconfig secrets.
	kubconfigSecretPrefix = "kubeconfig-"
)

// KubeconfigProvider is a struct that provides methods to interact with a kubeconfig cache.
type KubeconfigProvider struct {
	cache  *ttlcache.Cache[string, []byte]
	client v1.CoreV1Interface
	ttl    time.Duration
	logger *zap.SugaredLogger
	name   string
}

// New creates a new instance of KubeconfigProvider.
// It initializes the cache with the given TTL and loader function.
// The loader function is used to get the kubeconfig from the secret.
// name is used to identify the cache in the metrics.
func New(client v1.CoreV1Interface, logger *zap.SugaredLogger, ttl time.Duration, name string) *KubeconfigProvider {
	loader := loaderFunc(client, logger, ttl)

	return &KubeconfigProvider{
		client: client,
		cache: ttlcache.New[string, []byte](
			ttlcache.WithTTL[string, []byte](ttl),
			ttlcache.WithDisableTouchOnHit[string, []byte](),
			ttlcache.WithLoader[string, []byte](loader),
		),
		ttl:    ttl,
		logger: logger,
		name:   name,
	}
}

// loaderFunc returns a ttlcache.LoaderFunc that loads the kubeconfig from a Kubernetes secret.
// It logs the loading process and stores the kubeconfig in the cache with a TTL that includes jitter.
func loaderFunc(client v1.CoreV1Interface, logger *zap.SugaredLogger, ttl time.Duration) ttlcache.LoaderFunc[string, []byte] {
	return func(c *ttlcache.Cache[string, []byte], key string) *ttlcache.Item[string, []byte] {
		logger.Infof("loading Kubeconfig for: %v", key)

		kubeconfig, err := getKubeConfigFromSecret(logger, client, key)
		if err != nil {
			logger.Errorf("failed to get kubeconfig for runtimeID %s from secret: %s", key, err)
			return nil
		}

		logger.Infof("storing Kubeconfig for: %v", key)

		return c.Set(key, kubeconfig, getJitterTTL(ttl))
	}
}

// Get retrieves the kubeconfig for the given runtimeID from the cache.
// It cleans the cache from expired items and records the cache size in the metrics.
func (k *KubeconfigProvider) Get(runtimeID string) ([]byte, error) {
	k.cache.DeleteExpired()
	k.recordMetrics()

	if item := k.cache.Get(runtimeID); item != nil {
		return item.Value(), nil
	}

	return nil, fmt.Errorf("%w: %s", ErrNotFound, runtimeID)
}

// getKubeConfigFromSecret retrieves the kubeconfig from the secret.
func getKubeConfigFromSecret(logger *zap.SugaredLogger, client v1.CoreV1Interface, runtimeID string) ([]byte, error) {
	secretResourceName := kubconfigSecretPrefix + runtimeID

	secret, err := getKubeConfigSecret(logger, client, runtimeID, secretResourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	kubeconfig, found := secret.Data["config"]
	if !found {
		return nil, fmt.Errorf("kubeconfig-secret '%s' for runtime '%s' does not include the data-key 'config'",
			secretResourceName, runtimeID)
	}

	if len(kubeconfig) == 0 {
		return nil, fmt.Errorf("kubeconfig-secret '%s' for runtime '%s' includes an empty kubeconfig string",
			secretResourceName, runtimeID)
	}

	return kubeconfig, nil
}

// getKubeConfigSecret retrieves the kubeconfig secret from the cluster.
func getKubeConfigSecret(logger *zap.SugaredLogger, client v1.CoreV1Interface, runtimeID, secretResourceName string) (*corev1.Secret, error) {
	secret, err := client.Secrets(kcpNamespace).Get(context.Background(), secretResourceName, metav1.GetOptions{})
	if err != nil {
		switch {
		case k8serr.IsNotFound(err):
			logger.Debugf("kubeconfig provider cannot find kubeconfig-secret '%s' for runtimeID %s: %v", secretResourceName, runtimeID, err)
		case k8serr.IsForbidden(err):
			logger.Errorf("kubeconfig provider is not allowed to lookup kubeconfig-secret '%s' for runtimeID %s: %v", secretResourceName, runtimeID, err)
		default:
			logger.Errorf("kubeconfig provider failed to lookup kubeconfig-secret '%s' for runtimeID %s: %v", secretResourceName, runtimeID, err)
		}

		return nil, err
	}

	return secret, nil
}

// getJitterTTL returns a TTL with added jitter.
func getJitterTTL(ttl time.Duration) time.Duration {
	const jitterFactor = 0.3

	// If the TTL is less than 3 minutes, we don't add jitter.
	if ttl < 3*time.Minute {
		return ttl
	}

	buffer := int64(ttl.Minutes() * jitterFactor)
	jitter := rand.Int63n(buffer) + int64(ttl.Minutes())

	return time.Duration(jitter) * time.Minute
}
