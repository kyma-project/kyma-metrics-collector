package queue

import "k8s.io/client-go/util/workqueue"

func NewQueue(name string) workqueue.TypedDelayingInterface[string] {
	workqueue.SetProvider(&prometheusMetricsProvider{})
	queue := workqueue.NewTypedDelayingQueueWithConfig[string](
		workqueue.TypedDelayingQueueConfig[string]{
			Name: name,
		})

	return queue
}
