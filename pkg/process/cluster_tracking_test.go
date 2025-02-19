package process

import (
	"testing"
	"time"

	kebruntime "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/onsi/gomega"
)

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

func TestIsRuntimeTrackable(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	testCases := []struct {
		name     string
		runtime  kebruntime.RuntimeDTO
		expected bool
	}{
		{
			name: "should return true for successful provisioning",
			runtime: kebruntime.RuntimeDTO{
				Status: kebruntime.RuntimeStatus{
					Provisioning: &kebruntime.Operation{
						CreatedAt: time.Now(),
						State:     string(kebruntime.StateSucceeded),
					},
				},
			},
			expected: true,
		},
		{
			name: "should return false for suspension",
			runtime: kebruntime.RuntimeDTO{
				Status: kebruntime.RuntimeStatus{
					Suspension: &kebruntime.OperationsData{Data: []kebruntime.Operation{
						{CreatedAt: time.Now()},
					}},
				},
			},
			expected: false,
		},
		{
			name: "should return false for failing provisioning",
			runtime: kebruntime.RuntimeDTO{
				Status: kebruntime.RuntimeStatus{
					Provisioning: &kebruntime.Operation{
						CreatedAt: time.Now(),
						State:     string(kebruntime.StateFailed),
					},
				},
			},
			expected: false,
		},
		{
			name: "should return false for failing unsuspension",
			runtime: kebruntime.RuntimeDTO{
				Status: kebruntime.RuntimeStatus{
					Unsuspension: &kebruntime.OperationsData{Data: []kebruntime.Operation{
						{CreatedAt: time.Now(), State: string(kebruntime.StateFailed)},
					}},
				},
			},
			expected: false,
		},
		{
			name: "should return true for successful unsuspension",
			runtime: kebruntime.RuntimeDTO{
				Status: kebruntime.RuntimeStatus{
					Unsuspension: &kebruntime.OperationsData{Data: []kebruntime.Operation{
						{CreatedAt: time.Now(), State: string(kebruntime.StateSucceeded)},
					}},
				},
			},
			expected: true,
		},
		{
			name: "should return false for deprovisioning",
			runtime: kebruntime.RuntimeDTO{
				Status: kebruntime.RuntimeStatus{
					Deprovisioning: &kebruntime.Operation{CreatedAt: time.Now()},
				},
			},
			expected: false,
		},
		{
			name: "should return false for empty operations",
			runtime: kebruntime.RuntimeDTO{
				Status: kebruntime.RuntimeStatus{},
			},
			expected: false,
		},
		{
			name: "should return true for other operations",
			runtime: kebruntime.RuntimeDTO{
				Status: kebruntime.RuntimeStatus{
					Update: &kebruntime.OperationsData{Data: []kebruntime.Operation{
						{CreatedAt: time.Now()},
					}},
				},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isRuntimeTrackable(tc.runtime)
			g.Expect(result).To(gomega.Equal(tc.expected))
		})
	}
}
