package kubeconfigprovider

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var cacheSizeMetric = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Namespace: "kmc",
		Subsystem: "kubeconfig_cache",
		Name:      "size",
		Help:      "Number of items in the kubeconfig kubeconfigprovider.",
	}, []string{"name"})

func (k *KubeconfigProvider) recordMetrics() {
	cacheSizeMetric.With(prometheus.Labels{"name": k.name}).Set(float64(k.cache.Len()))
}
