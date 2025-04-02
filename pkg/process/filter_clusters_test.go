package process

import (
	"github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestParseClusterToBeFiltered(t *testing.T) {
	tt := []struct {
		name     string
		cluster  string
		expected map[string]struct{}
	}{
		{
			name: "file with single cluster",
			cluster: `meteringAccounts: 
                       - "cluster-1"`,
			expected: map[string]struct{}{"cluster-1": {}},
		},
		{
			name: "with duplicate cluster names",
			cluster: `meteringAccounts:
                       - "cluster-1"
                       - "cluster-1"`,
			expected: map[string]struct{}{"cluster-1": {}},
		},
		{
			name: "with multiple clusters",
			cluster: `meteringAccounts:
                       - "cluster-1"
                       - "cluster-2"`,
			expected: map[string]struct{}{"cluster-1": {}, "cluster-2": {}},
		},
		{
			name:     "empty cluster list",
			cluster:  ``,
			expected: map[string]struct{}{},
		},
		{
			name:     "with invalid YAML",
			cluster:  `"invalid: yaml"`,
			expected: nil,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := parseRuntimesToBeFiltered([]byte(tc.cluster))
			if tc.expected == nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, actual)
			}
		})
	}
}

func TestSkipRuntime(t *testing.T) {
	tt := []struct {
		name     string
		runtime  runtime.RuntimeDTO
		filter   map[string]struct{}
		expected bool
	}{
		{
			name: "runtime in filter list",
			runtime: runtime.RuntimeDTO{
				GlobalAccountID: "cluster-1",
			},
			filter:   map[string]struct{}{"cluster-1": {}},
			expected: true,
		},
		{
			name: "runtime not in filter list",
			runtime: runtime.RuntimeDTO{
				GlobalAccountID: "cluster-1",
			},
			filter:   map[string]struct{}{"cluster-2": {}},
			expected: false,
		},
		{
			name: "empty filter list",
			runtime: runtime.RuntimeDTO{
				GlobalAccountID: "cluster-1",
			},
			filter:   map[string]struct{}{},
			expected: false,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			actual := skipRuntime(tc.runtime, tc.filter)
			require.Equal(t, tc.expected, actual)
		})
	}
}
