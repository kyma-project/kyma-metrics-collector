package process

import (
	kebruntime "github.com/kyma-project/kyma-environment-broker/common/runtime"
	kmctesting "github.com/kyma-project/kyma-metrics-collector/pkg/testing"
	"github.com/onsi/gomega"
	"testing"
	"time"
)

// test sortOperations
// TestSortOperations tests the sortOperations function.
func TestSortOperations(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	testCases := []struct {
		name        string
		runtime     kebruntime.RuntimeDTO
		expected    []runtimeState
		expectedLen int
	}{
		{
			name: "should sort operations by time",
			runtime: kebruntime.RuntimeDTO{
				Status: kebruntime.RuntimeStatus{
					Provisioning: &kebruntime.Operation{
						CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
					},
					Deprovisioning: &kebruntime.Operation{
						CreatedAt: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
					},
					UpgradingKyma: &kebruntime.OperationsData{Data: []kebruntime.Operation{
						{CreatedAt: time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC)},
					}},
					UpgradingCluster: &kebruntime.OperationsData{Data: []kebruntime.Operation{
						{CreatedAt: time.Date(2023, 1, 4, 0, 0, 0, 0, time.UTC)},
					}},
					Update: &kebruntime.OperationsData{Data: []kebruntime.Operation{
						{CreatedAt: time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC)},
					}},
					Suspension: &kebruntime.OperationsData{Data: []kebruntime.Operation{
						{CreatedAt: time.Date(2023, 1, 6, 0, 0, 0, 0, time.UTC)},
					}},
					Unsuspension: &kebruntime.OperationsData{Data: []kebruntime.Operation{
						{CreatedAt: time.Date(2023, 1, 7, 0, 0, 0, 0, time.UTC)},
					}},
				},
			},
			expected:    []runtimeState{provisioning, deprovisioning, upgradingkyma, upgradingcluster, update, suspension, unsuspension},
			expectedLen: 7,
		},
		{
			name: "should handle multiple suspensions and following unsuspensions",
			runtime: kebruntime.RuntimeDTO{
				Status: kebruntime.RuntimeStatus{
					Suspension: &kebruntime.OperationsData{Data: []kebruntime.Operation{
						{CreatedAt: time.Date(2023, 1, 6, 0, 0, 0, 0, time.UTC)},
						{CreatedAt: time.Date(2023, 1, 8, 0, 0, 0, 0, time.UTC)},
					}},
					Unsuspension: &kebruntime.OperationsData{Data: []kebruntime.Operation{
						{CreatedAt: time.Date(2023, 1, 7, 0, 0, 0, 0, time.UTC)},
						{CreatedAt: time.Date(2023, 1, 9, 0, 0, 0, 0, time.UTC)},
					}},
				},
			},
			expected:    []runtimeState{suspension, unsuspension, suspension, unsuspension},
			expectedLen: 4,
		},
		{
			name: "should handle empty operations",
			runtime: kebruntime.RuntimeDTO{
				Status: kebruntime.RuntimeStatus{},
			},
			expected:    []runtimeState{},
			expectedLen: 0,
		},
		{
			name: "should handle nil operation times",
			runtime: kebruntime.RuntimeDTO{
				Status: kebruntime.RuntimeStatus{
					Provisioning:   &kebruntime.Operation{},
					Deprovisioning: &kebruntime.Operation{},
				},
			},
			expected:    []runtimeState{provisioning, deprovisioning},
			expectedLen: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			operations := sortOperations(tc.runtime)

			g.Expect(operations).To(gomega.HaveLen(tc.expectedLen))
			for i, op := range operations {
				g.Expect(op.state).To(gomega.Equal(tc.expected[i]))
			}
		})
	}
}

func TestIsTrackableBasedOnOperations(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	testCases := []struct {
		name        string
		runtime     kebruntime.RuntimeDTO
		isTrackable bool
	}{
		{
			name: "should return false when no operations exist",
			runtime: kebruntime.RuntimeDTO{
				Status: kebruntime.RuntimeStatus{},
			},
			isTrackable: false,
		},
		{
			name: "should return false when last operation is suspension",
			runtime: kebruntime.RuntimeDTO{
				Status: kebruntime.RuntimeStatus{
					Suspension: &kebruntime.OperationsData{Data: []kebruntime.Operation{
						{CreatedAt: time.Date(2023, 1, 6, 0, 0, 0, 0, time.UTC)},
					}},
				},
			},
			isTrackable: false,
		},
		{
			name: "should return false when last operation is deprovisioning",
			runtime: kebruntime.RuntimeDTO{
				Status: kebruntime.RuntimeStatus{
					Deprovisioning: &kebruntime.Operation{CreatedAt: time.Date(2023, 1, 6, 0, 0, 0, 0, time.UTC)},
				},
			},
			isTrackable: false,
		},
		{
			name: "should return true when last operation is not suspension or deprovisioning",
			runtime: kebruntime.RuntimeDTO{
				Status: kebruntime.RuntimeStatus{
					Provisioning: &kebruntime.Operation{CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
					Update: &kebruntime.OperationsData{Data: []kebruntime.Operation{
						{CreatedAt: time.Date(2023, 1, 5, 0, 0, 0, 0, time.UTC)},
					}},
				},
			},
			isTrackable: true,
		},
		{
			name: "should handle multiple suspensions and following unsuspensions",
			runtime: kebruntime.RuntimeDTO{
				Status: kebruntime.RuntimeStatus{
					Suspension: &kebruntime.OperationsData{Data: []kebruntime.Operation{
						{CreatedAt: time.Date(2023, 1, 6, 0, 0, 0, 0, time.UTC)},
						{CreatedAt: time.Date(2023, 1, 8, 0, 0, 0, 0, time.UTC)},
					}},
					Unsuspension: &kebruntime.OperationsData{Data: []kebruntime.Operation{
						{CreatedAt: time.Date(2023, 1, 7, 0, 0, 0, 0, time.UTC)},
						{CreatedAt: time.Date(2023, 1, 9, 0, 0, 0, 0, time.UTC)},
					}},
				},
			},
			isTrackable: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isTrackable := isTrackableBasedOnOperations(tc.runtime)
			g.Expect(isTrackable).To(gomega.Equal(tc.isTrackable))
		})
	}
}

func TestIsTrackableState(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	// t.Parallel()

	// test cases
	testCases := []struct {
		name              string
		givenRuntimeState kebruntime.State
		expectedBool      bool
	}{
		{
			name:              "should return true when shoot is in succeeded state",
			givenRuntimeState: kebruntime.StateSucceeded,
			expectedBool:      true,
		},
		{
			name:              "should return true when shoot is in error state",
			givenRuntimeState: kebruntime.StateError,
			expectedBool:      true,
		},
		{
			name:              "should return true when shoot is in upgrading state",
			givenRuntimeState: kebruntime.StateUpgrading,
			expectedBool:      true,
		},
		{
			name:              "should return true when shoot is in updating state",
			givenRuntimeState: kebruntime.StateUpdating,
			expectedBool:      true,
		},
		{
			name:              "should return false when shoot is in deprovisioned state",
			givenRuntimeState: kebruntime.StateDeprovisioned,
			expectedBool:      false,
		},
		{
			name:              "should return false when shoot is in deprovisioned incomplete state",
			givenRuntimeState: kebruntime.StateDeprovisionIncomplete,
			expectedBool:      false,
		},
		{
			name:              "should return false when shoot is in deprovisioning  state",
			givenRuntimeState: kebruntime.StateDeprovisioning,
			expectedBool:      false,
		},
		{
			name:              "should return false when shoot is in failed state",
			givenRuntimeState: kebruntime.StateFailed,
			expectedBool:      false,
		},
		{
			name:              "should return false when shoot is in suspended state",
			givenRuntimeState: kebruntime.StateSuspended,
			expectedBool:      false,
		},
		{
			name:              "should return false when shoot is in provisioning state",
			givenRuntimeState: kebruntime.StateProvisioning,
			expectedBool:      false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			isTrackable := isTrackableState(tc.givenRuntimeState)

			// then
			g.Expect(isTrackable).To(gomega.Equal(tc.expectedBool))
		})
	}
}

func TestIsRuntimeTrackable(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	// t.Parallel()

	// const used in all test cases
	subAccountID := "c7db696a-32fa-48ee-9009-aa3e0034121e"
	shootName := "shoot-gKtxg"

	// test cases
	testCases := []struct {
		name         string
		givenRuntime kebruntime.RuntimeDTO
		isTrackable  bool
	}{
		{
			name:         "should return true when runtime is in trackable state and provisioned status",
			givenRuntime: kmctesting.NewRuntimesDTO(subAccountID, shootName, kmctesting.WithProvisioningSucceededStatus(kebruntime.StateSucceeded)),
			isTrackable:  true,
		},
		//{
		//	name:         "should return true when runtime is in trackable state and deprovisioned status",
		//	givenRuntime: kmctesting.NewRuntimesDTO(subAccountID, shootName, kmctesting.WithProvisionedAndDeprovisionedStatus(kebruntime.StateSucceeded)),
		//	isTrackable:  true,
		//},
		{
			name:         "should return false when runtime is in non-trackable state and provisioned status",
			givenRuntime: kmctesting.NewRuntimesDTO(subAccountID, shootName, kmctesting.WithProvisioningSucceededStatus(kebruntime.StateDeprovisioning)),
			isTrackable:  false,
		},
		{
			name:         "should return false when runtime is in non-trackable state and deprovisioned status",
			givenRuntime: kmctesting.NewRuntimesDTO(subAccountID, shootName, kmctesting.WithProvisionedAndDeprovisionedStatus(kebruntime.StateDeprovisioning)),
			isTrackable:  false,
		},
		{
			name:         "should return false when runtime state has status deprovisioning",
			givenRuntime: kmctesting.NewRuntimesDTO(subAccountID, shootName, kmctesting.WithState(kebruntime.StateDeprovisioning)),
			isTrackable:  false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			isRuntimeTrackable := isRuntimeTrackable(tc.givenRuntime)

			// then
			g.Expect(isRuntimeTrackable).To(gomega.Equal(tc.isTrackable))
		})
	}
}
