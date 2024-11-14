package cache

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var cacheSizeMetric = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Namespace: "kmc",
		Subsystem: "kubeconfig_cache",
		Name:      "size",
		Help:      "Number of items in the kubeconfig cache.",
	}, nil)

func recordMetrics() {
	cacheSizeMetric.With(prometheus.Labels{}).Set(float64(kubeConfigCache.Len()))
}
