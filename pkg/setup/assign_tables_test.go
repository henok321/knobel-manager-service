package setup

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestAssignTables(t *testing.T) {
	type args struct {
		teams     map[int][]int
		tableSize int
		teamSize  int
		seed      int64
	}

	tests := []struct {
		name     string
		args     args
		expected string
		err      bool
	}{
		{
			name: "assign tables with seed 1",
			args: args{
				teams: map[int][]int{
					1: {1, 2, 3, 4},
					2: {5, 6, 7, 8},
					3: {9, 10, 11, 12},
					4: {13, 14, 15, 16},
					5: {17, 18, 19, 20},
					6: {21, 22, 23, 24},
					7: {25, 26, 27, 28},
					8: {29, 30, 31, 32},
				},
				tableSize: 4,
				teamSize:  4,
				seed:      1,
			},
			expected: "expected_1.json",
			err:      false,
		},
		{
			name: "assign tables with seed 2",
			args: args{
				teams: map[int][]int{
					1: {1, 2, 3, 4},
					2: {5, 6, 7, 8},
					3: {9, 10, 11, 12},
					4: {13, 14, 15, 16},
					5: {17, 18, 19, 20},
					6: {21, 22, 23, 24},
					7: {25, 26, 27, 28},
					8: {29, 30, 31, 32},
				},
				tableSize: 4,
				teamSize:  4,
				seed:      2,
			},
			expected: "expected_2.json",
			err:      false,
		},
		{
			name: "assign tables invalid table size",
			args: args{
				teams: map[int][]int{
					1: {1, 2, 3, 4},
					2: {5, 6, 7, 8},
					3: {9, 10, 11, 12},
					4: {13, 14, 15, 16},
					5: {17, 18, 19, 20},
					6: {21, 22, 23, 24},
					7: {25, 26, 27, 28},
					8: {29, 30, 31, 32},
				},
				tableSize: 5,
				teamSize:  4,
				seed:      1,
			},
			err: true,
		},
		{
			name: "assign tables invalid team size",
			args: args{
				teams: map[int][]int{
					1: {1, 2, 3, 4},
					2: {5, 6, 7, 8},
					3: {9, 10, 11, 12},
					4: {13, 14, 15, 16},
					5: {17, 18, 19, 20},
					6: {21, 22, 23, 24},
					7: {25, 26, 27, 28},
					8: {29, 30, 31, 32},
				},
				tableSize: 4,
				teamSize:  5,
				seed:      1,
			},
			err: true,
		},
		{
			name: "assign tables invalid team assignment",
			args: args{
				teams: map[int][]int{
					1: {1, 2, 3, 4},
					2: {5, 6, 7},
					3: {9, 10, 11, 12},
					4: {13, 14, 15, 16},
					5: {17, 18, 19, 20},
					6: {21, 22, 23, 24},
					7: {25, 26, 27, 28},
					8: {29, 30, 31, 32},
				},
				tableSize: 5,
				teamSize:  4,
				seed:      1,
			},
			err: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AssignTables(TeamSetup{Teams: tt.args.teams, TeamSize: tt.args.teamSize, TableSize: tt.args.tableSize}, tt.args.seed)

			if tt.err {
				require.Error(t, err, "Should fail because of expected error")
			} else {
				require.NoError(t, err, "Assignment should not throw error")
				gotJSON, err := json.Marshal(got)

				require.NoError(t, err, "Could not parse result to json")

				expectedJSON, err := os.ReadFile(tt.expected)
				require.NoError(t, err, "Could not read expected json")

				assert.JSONEq(t, string(expectedJSON), string(gotJSON))
			}
		})
	}
}
