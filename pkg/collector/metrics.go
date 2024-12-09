package collector

import (
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"strconv"
)

const (
	namespace          = "kmc"
	subsystem          = "collector"
	shootNameLabel     = "shoot_name"
	instanceIdLabel    = "instance_id"
	runtimeIdLabel     = "runtime_id"
	subAccountLabel    = "sub_account_id"
	globalAccountLabel = "global_account_id"
	successLabel       = "success"
	resourceNameLabel  = "resource_name"
	backendNameLabel   = "backend_name"
	EDPBackendName     = "edp"
)

var (
	TotalScans = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "scans_total",
			Help:      "Total number of scans for each billable resource in SKR.",
		},
		[]string{successLabel, resourceNameLabel, shootNameLabel, instanceIdLabel, runtimeIdLabel, subAccountLabel, globalAccountLabel},
	)

	TotalScansConverted = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "scans_converted_total",
			Help:      "Total number of scans converted to the measurement required by the backend.",
		},
		[]string{successLabel, resourceNameLabel, backendNameLabel, shootNameLabel, instanceIdLabel, runtimeIdLabel, subAccountLabel, globalAccountLabel},
	)
)

func RecordScan(success bool, resourceName string, runtimeInfo runtime.Info) {
	// the order of the values should be same as defined in the metric declaration.
	TotalScans.WithLabelValues(
		strconv.FormatBool(success),
		resourceName,
		runtimeInfo.ShootName,
		runtimeInfo.InstanceID,
		runtimeInfo.RuntimeID,
		runtimeInfo.SubAccountID,
		runtimeInfo.GlobalAccountID,
	).Inc()
}

func RecordScanConversion(success bool, resourceName string, backendName string, runtimeInfo runtime.Info) {
	// the order of the values should be same as defined in the metric declaration.
	TotalScansConverted.WithLabelValues(
		strconv.FormatBool(success),
		resourceName,
		backendName,
		runtimeInfo.ShootName,
		runtimeInfo.InstanceID,
		runtimeInfo.RuntimeID,
		runtimeInfo.SubAccountID,
		runtimeInfo.GlobalAccountID,
	).Inc()
}
