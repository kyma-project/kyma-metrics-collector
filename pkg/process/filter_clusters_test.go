package process

import (
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
                       - "8946D5DE-E59A-4F4D-B65D-4595758D1FB1"`,
			expected: map[string]struct{}{"8946D5DE-E59A-4F4D-B65D-4595758D1FB1": {}},
		},
		{
			name: "with duplicate cluster names",
			cluster: `meteringAccounts:
                       - "8946D5DE-E59A-4F4D-B65D-4595758D1FB1"
                       - "8946D5DE-E59A-4F4D-B65D-4595758D1FB1"`,
			expected: map[string]struct{}{"8946D5DE-E59A-4F4D-B65D-4595758D1FB1": {}},
		},
		{
			name: "with multiple clusters",
			cluster: `meteringAccounts:
                       - "8946D5DE-E59A-4F4D-B65D-4595758D1FB1"
                       - "E653F9B0-97F1-4BF4-AAF2-268C5217CF49"`,
			expected: map[string]struct{}{
				"8946D5DE-E59A-4F4D-B65D-4595758D1FB1": {},
				"E653F9B0-97F1-4BF4-AAF2-268C5217CF49": {}},
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
