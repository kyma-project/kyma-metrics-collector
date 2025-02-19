package process

import (
	kebruntime "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"sort"
	"time"
)

type runtimeState int

const (
	provisioning runtimeState = iota
	deprovisioning
	upgradingkyma
	upgradingcluster
	update
	suspension
	unsuspension
)

type simpleOperation struct {
	state runtimeState
	time  time.Time
}

func isRuntimeTrackable(runtime kebruntime.RuntimeDTO) bool {
	switch runtime.Status.State {
	case kebruntime.StateDeprovisioned, kebruntime.StateDeprovisioning, kebruntime.StateSuspended:
		return false
	}

	trackable := isTrackableBasedOnOperations(runtime)

	// we are certain that the runtime is not trackable only if this function returns false
	if !trackable {
		return false
	}

	return isTrackableState(runtime.Status.State)
}

func isTrackableBasedOnOperations(runtime kebruntime.RuntimeDTO) bool {
	operations := sortOperations(runtime)

	// if a cluster does not have any operations, it is not trackable
	if len(operations) == 0 {
		return false
	}

	// if the last operation is suspension or deprovisioning, the runtime is not trackable
	lastOperation := operations[len(operations)-1]
	if lastOperation.state == suspension || lastOperation.state == deprovisioning {
		return false
	}
	return true
}

func sortOperations(runtime kebruntime.RuntimeDTO) []simpleOperation {
	var operations []simpleOperation
	if runtime.Status.Provisioning != nil {
		operations = append(operations, simpleOperation{state: provisioning, time: runtime.Status.Provisioning.CreatedAt})
	}
	if runtime.Status.Deprovisioning != nil {
		operations = append(operations, simpleOperation{state: deprovisioning, time: runtime.Status.Deprovisioning.CreatedAt})
	}
	if runtime.Status.UpgradingKyma != nil {
		for _, op := range runtime.Status.UpgradingKyma.Data {
			operations = append(operations, simpleOperation{state: upgradingkyma, time: op.CreatedAt})
		}
	}
	if runtime.Status.UpgradingCluster != nil {
		for _, op := range runtime.Status.UpgradingCluster.Data {
			operations = append(operations, simpleOperation{state: upgradingcluster, time: op.CreatedAt})
		}
	}
	if runtime.Status.Update != nil {
		for _, op := range runtime.Status.Update.Data {
			operations = append(operations, simpleOperation{state: update, time: op.CreatedAt})
		}
	}
	if runtime.Status.Suspension != nil {
		for _, op := range runtime.Status.Suspension.Data {
			operations = append(operations, simpleOperation{state: suspension, time: op.CreatedAt})
		}
	}
	if runtime.Status.Unsuspension != nil {
		for _, op := range runtime.Status.Unsuspension.Data {
			operations = append(operations, simpleOperation{state: unsuspension, time: op.CreatedAt})
		}
	}

	// sort operations by time
	sort.Slice(operations, func(i, j int) bool {
		return operations[i].time.Before(operations[j].time)
	})
	return operations
}

// isTrackableState returns true if the runtime state is trackable, otherwise returns false.
func isTrackableState(state kebruntime.State) bool {
	//nolint:exhaustive // we only care about these states
	switch state {
	case kebruntime.StateSucceeded, kebruntime.StateError, kebruntime.StateUpgrading, kebruntime.StateUpdating:
		return true
	}

	return false
}
