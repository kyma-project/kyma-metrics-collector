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

func isRuntimeTrackable(runtime kebruntime.RuntimeDTO) bool {
	// we assume the following:
	// a successful	provisioning operation marks the cluster as billable
	// any suspension operation marks the cluster as not billable
	// a successful unsuspension operation marks the cluster as billable again
	// cluster lifecycle ends with deprovisioning
	// any other operation assumes that the cluster was successfully provisioned and is billable
	//
	// we only check the last operation for these cases
	//
	operations := sortOperations(runtime)

	// if a cluster does not have any operations, it is not trackable
	if len(operations) == 0 {
		return false
	}

	lastOperation := operations[len(operations)-1]

	//nolint:exhaustive // we only care about the for states, not the others
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
