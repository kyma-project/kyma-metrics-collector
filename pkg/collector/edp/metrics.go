package edp

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	namespace = "kmc"
	subsystem = "edp"
	// responseCodeLabel name of the status code labels used by multiple metrics.
	responseCodeLabel = "status"
	// requestURLLabel name of the request URL label used by multiple metrics.
	requestURLLabel = "request_url"
	// metrics names.
	latencyMetricName = "request_duration_seconds"
)

var latencyMetric = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      latencyMetricName,
		Help:      "Duration of HTTP request to EDP in seconds.",
		Buckets:   []float64{0.01, 0.02, 0.05, 0.1, 0.25, 0.5, 1, 2},
	},
	[]string{responseCodeLabel, requestURLLabel},
)

func recordEDPLatency(duration time.Duration, statusCode int, destSvc string) {
	// the order of the values should be same as defined in the metric declaration.
	latencyMetric.WithLabelValues(fmt.Sprint(statusCode), destSvc).Observe(duration.Seconds())
}
