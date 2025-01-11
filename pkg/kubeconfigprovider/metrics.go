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
	}, nil)

func recordMetrics() {
	// TODO: Implement metrics recording
	// cacheSizeMetric.With(prometheus.Labels{}).Set(float64(kubeConfigCache.Len()))
}
