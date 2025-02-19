package process

import (
	"sort"
	"time"

	kebruntime "github.com/kyma-project/kyma-environment-broker/common/runtime"
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
	state     runtimeState
	time      time.Time
	succeeded bool
}

// isRuntimeTrackable determines if a runtime is trackable based on its operations.
// A runtime is considered trackable if:
// - It has a successful provisioning or unsuspension operation.
// - It does not have a suspension or deprovisioning operation as the last operation.
// - It has any other operation assuming the cluster was successfully provisioned and is billable.
func isRuntimeTrackable(runtime kebruntime.RuntimeDTO) bool {
	// Sort operations by time
	operations := sortOperations(runtime)

	// If a cluster does not have any operations, it is not trackable
	if len(operations) == 0 {
		return false
	}

	// Get the last operation
	lastOperation := operations[len(operations)-1]

	// Determine trackability based on the last operation state
	//nolint:exhaustive // we only care about the four states, not the others
	switch lastOperation.state {
	case provisioning, unsuspension:
		return lastOperation.succeeded
	case suspension, deprovisioning:
		return false
	default:
		return true
	}
}

func newSimpleOperation(state runtimeState, operation kebruntime.Operation) simpleOperation {
	return simpleOperation{
		state:     state,
		time:      operation.CreatedAt,
		succeeded: operationSucceeded(operation),
	}
}

func operationSucceeded(operation kebruntime.Operation) bool {
	return operation.State == string(kebruntime.StateSucceeded)
}

func sortOperations(runtime kebruntime.RuntimeDTO) []simpleOperation {
	var operations []simpleOperation
	if runtime.Status.Provisioning != nil {
		operations = append(operations, newSimpleOperation(provisioning, *runtime.Status.Provisioning))
	}

	if runtime.Status.Deprovisioning != nil {
		operations = append(operations, newSimpleOperation(deprovisioning, *runtime.Status.Deprovisioning))
	}

	if runtime.Status.UpgradingKyma != nil {
		for _, op := range runtime.Status.UpgradingKyma.Data {
			operations = append(operations, newSimpleOperation(upgradingkyma, op))
		}
	}

	if runtime.Status.UpgradingCluster != nil {
		for _, op := range runtime.Status.UpgradingCluster.Data {
			operations = append(operations, newSimpleOperation(upgradingcluster, op))
		}
	}

	if runtime.Status.Update != nil {
		for _, op := range runtime.Status.Update.Data {
			operations = append(operations, newSimpleOperation(update, op))
		}
	}

	if runtime.Status.Suspension != nil {
		for _, op := range runtime.Status.Suspension.Data {
			operations = append(operations, newSimpleOperation(suspension, op))
		}
	}

	if runtime.Status.Unsuspension != nil {
		for _, op := range runtime.Status.Unsuspension.Data {
			operations = append(operations, newSimpleOperation(unsuspension, op))
		}
	}

	// sort operations by time
	sort.Slice(operations, func(i, j int) bool {
		return operations[i].time.Before(operations[j].time)
	})

	return operations
}
