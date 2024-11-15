package queue

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"k8s.io/client-go/util/workqueue"
)

type prometheusMetricsProvider struct{}

const (
	namespace                  = "kmc"
	subsystem                  = "workqueue"
	depthKey                   = "depth"
	addsKey                    = "adds_total"
	queueLatencyKey            = "queue_duration_seconds"
	workDurationKey            = "work_duration_seconds"
	unfinishedWorkKey          = "unfinished_work_seconds"
	longestRunningProcessorKey = "longest_running_processor_seconds"
	retriesKey                 = "retries_total"
	bucketFactor               = 10
	bucketCount                = 10
)

var (
	smallestBucket = 10 * time.Nanosecond.Seconds() // 10ns

	depth = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   namespace,
		Subsystem:   subsystem,
		Name:        depthKey,
		Help:        "Current depth of workqueue.",
		ConstLabels: nil,
	}, []string{"name"})

	adds = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      addsKey,
		Help:      "Total number of adds handled by workqueue.",
	}, []string{"name"})

	latency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      queueLatencyKey,
		Help:      "Amount of time (in seconds) when an item stays in workqueue before being requested.",
		Buckets:   prometheus.ExponentialBuckets(smallestBucket, bucketFactor, bucketCount),
	}, []string{"name"})

	workDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      workDurationKey,
		Help:      "Amount of time (in seconds) taken for processing an item from workqueue.",
		Buckets:   prometheus.ExponentialBuckets(smallestBucket, bucketFactor, bucketCount),
	}, []string{"name"})

	unfinished = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      unfinishedWorkKey,
		Help:      "Amount of time (in seconds) already taken by the work in progress that hasn't been observed by **work_duration**. Large values indicate stuck threads. You can deduce the number of stuck threads by observing the rate at which the metric increases.",
	}, []string{"name"})

	longestRunningProcessor = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      longestRunningProcessorKey,
		Help:      "Amount of time (in seconds) taken by the longest running processor for workqueue.",
	}, []string{"name"})

	retries = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      retriesKey,
		Help:      "Total number of retries handled by workqueue.",
	}, []string{"name"})
)

func (p prometheusMetricsProvider) NewDepthMetric(name string) workqueue.GaugeMetric {
	return depth.WithLabelValues(name)
}

func (p prometheusMetricsProvider) NewAddsMetric(name string) workqueue.CounterMetric {
	return adds.WithLabelValues(name)
}

func (p prometheusMetricsProvider) NewLatencyMetric(name string) workqueue.HistogramMetric {
	return latency.WithLabelValues(name)
}

func (p prometheusMetricsProvider) NewWorkDurationMetric(name string) workqueue.HistogramMetric {
	return workDuration.WithLabelValues(name)
}

func (p prometheusMetricsProvider) NewUnfinishedWorkSecondsMetric(name string) workqueue.SettableGaugeMetric {
	return unfinished.WithLabelValues(name)
}

func (p prometheusMetricsProvider) NewLongestRunningProcessorSecondsMetric(name string) workqueue.SettableGaugeMetric {
	return longestRunningProcessor.WithLabelValues(name)
}

func (p prometheusMetricsProvider) NewRetriesMetric(name string) workqueue.CounterMetric {
	return retries.WithLabelValues(name)
}

var _ workqueue.MetricsProvider = &prometheusMetricsProvider{}
