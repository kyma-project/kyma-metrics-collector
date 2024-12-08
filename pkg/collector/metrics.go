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
)

var (
	TotalScans = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "scans_total",
			Help:      "Total number of scans for resources in SKR.",
		},
		[]string{successLabel, resourceNameLabel, shootNameLabel, instanceIdLabel, runtimeIdLabel, subAccountLabel, globalAccountLabel},
	)

	TotalScansConversionsToEDP = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "scans_converted_to_edp_total",
			Help:      "Total number of scans converted to EDP measurements.",
		},
		[]string{successLabel, resourceNameLabel, shootNameLabel, instanceIdLabel, runtimeIdLabel, subAccountLabel, globalAccountLabel},
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

func RecordScanConversionToEDP(success bool, resourceName string, runtimeInfo runtime.Info) {
	// the order of the values should be same as defined in the metric declaration.
	TotalScansConversionsToEDP.WithLabelValues(
		strconv.FormatBool(success),
		resourceName,
		runtimeInfo.ShootName,
		runtimeInfo.InstanceID,
		runtimeInfo.RuntimeID,
		runtimeInfo.SubAccountID,
		runtimeInfo.GlobalAccountID,
	).Inc()
}
