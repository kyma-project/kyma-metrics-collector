package process

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseClusterToBeFiltered(t *testing.T) {
	tt := []struct {
		name        string
		cluster     string
		expected    map[string]struct{}
		expectedErr error
	}{
		{
			name: "file with single cluster",
			cluster: `globalAccounts: 
                       - "8946D5DE-E59A-4F4D-B65D-4595758D1FB1"`,
			expected:    map[string]struct{}{"8946D5DE-E59A-4F4D-B65D-4595758D1FB1": {}},
			expectedErr: nil,
		},
		{
			name: "with duplicate cluster names",
			cluster: `globalAccounts:
                       - "8946D5DE-E59A-4F4D-B65D-4595758D1FB1"
                       - "8946D5DE-E59A-4F4D-B65D-4595758D1FB1"`,
			expected:    map[string]struct{}{"8946D5DE-E59A-4F4D-B65D-4595758D1FB1": {}},
			expectedErr: nil,
		},
		{
			name: "with multiple clusters",
			cluster: `globalAccounts:
                       - "8946D5DE-E59A-4F4D-B65D-4595758D1FB1"
                       - "E653F9B0-97F1-4BF4-AAF2-268C5217CF49"`,
			expected: map[string]struct{}{
				"8946D5DE-E59A-4F4D-B65D-4595758D1FB1": {},
				"E653F9B0-97F1-4BF4-AAF2-268C5217CF49": {},
			},
			expectedErr: nil,
		},
		{
			name: "with invalid global account ID",
			cluster: `globalAccounts:
                       - "a8946D5DE-E59A-4F4D-B65D-4595758D1FB1"
                       - "bE653F9B0-97F1-4BF4-AAF2-268C5217CF49"`,
			expected:    nil,
			expectedErr: fmt.Errorf("invalid global account IDs: a8946D5DE-E59A-4F4D-B65D-4595758D1FB1, bE653F9B0-97F1-4BF4-AAF2-268C5217CF49"),
		},
		{
			name:        "empty cluster list",
			cluster:     ``,
			expected:    map[string]struct{}{},
			expectedErr: nil,
		},
		{
			name:        "with invalid YAML",
			cluster:     `"invalid: yaml"`,
			expected:    nil,
			expectedErr: fmt.Errorf("yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `invalid...` into process.accounts"),
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := parseRuntimesToBeFiltered([]byte(tc.cluster))
			if tc.expectedErr != nil {
				require.Error(t, err)
				require.Equal(t, tc.expectedErr.Error(), err.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, actual)
			}
		})
	}
}
