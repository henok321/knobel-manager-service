package setup

import (
	"encoding/json"
	"os"
	"testing"

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
		name string
		args args
		want string
		err  bool
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
			want: "expected_1.json",
			err:  false,
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
			want: "expected_2.json",
			err:  false,
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
			got, err := AssignTables(tt.args.teams, tt.args.teamSize, tt.args.tableSize, tt.args.seed)
			if tt.err {
				assert.Error(t, err, "Should fail because of expected error")
			} else {

				assert.NoError(t, err, "Assignment should not throw error")
				gotJson, err := json.Marshal(got)

				assert.NoError(t, err, "Could not parse result to json")

				expectedJson, err := os.ReadFile(tt.want)
				assert.NoError(t, err, "Could not read expected json")

				assert.JSONEq(t, string(expectedJson), string(gotJson))
			}

		})
	}
}
