package setup

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// Regression: teamSize a strict multiple of tableSize yields more tables than
// teams; the assignment loop used to fill only len(teams) tables and retry forever.
func TestAssignTablesMoreTablesThanTeams(t *testing.T) {
	teams := map[int][]int{
		1: {1, 2, 3, 4, 5, 6, 7, 8},
		2: {9, 10, 11, 12, 13, 14, 15, 16},
		3: {17, 18, 19, 20, 21, 22, 23, 24},
		4: {25, 26, 27, 28, 29, 30, 31, 32},
	}

	got, err := AssignTables(TeamSetup{Teams: teams, TeamSize: 8, TableSize: 4}, 1)

	require.NoError(t, err)
	require.Len(t, got, 8, "32 players / table size 4 = 8 tables")

	assignedPlayers := map[int]bool{}

	for _, players := range got {
		assert.Len(t, players, 4, "every table must be full")

		teamsAtTable := map[int]bool{}

		for _, p := range players {
			assert.False(t, teamsAtTable[p.TeamID], "no two players of the same team at one table")
			teamsAtTable[p.TeamID] = true
			assert.False(t, assignedPlayers[p.ID], "player %d assigned twice", p.ID)
			assignedPlayers[p.ID] = true
		}
	}

	assert.Len(t, assignedPlayers, 32, "all players must be assigned exactly once")
}
