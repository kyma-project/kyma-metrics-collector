package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"k8s.io/client-go/tools/metrics"
)

const (
	namespace   = "kmc"
	subsystem   = "tls_cache"
	createKey   = "create_total"
	cacheLenKey = "entries_total"
)

var (
	creates = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      createKey,
		Help:      "Total number of create calls.",
	}, []string{"result"})

	entries = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      cacheLenKey,
		Help:      "Total number of entries in the cache.",
	}, []string{})
)

type TLSCacheMetrics struct{}

func (T TLSCacheMetrics) Increment(result string) {
	creates.WithLabelValues(result).Inc()
}

func (T TLSCacheMetrics) Observe(value int) {
	entries.WithLabelValues().Set(float64(value))
}

var (
	_ metrics.TransportCacheMetric       = &TLSCacheMetrics{}
	_ metrics.TransportCreateCallsMetric = &TLSCacheMetrics{}
)

func RegisterTLSCacheMetrics() {
	opts := metrics.RegisterOpts{
		TransportCacheEntries: TLSCacheMetrics{},
		TransportCreateCalls:  TLSCacheMetrics{},
	}

	metrics.Register(opts)
}
