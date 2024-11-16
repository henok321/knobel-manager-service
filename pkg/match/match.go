package match

import (
	"fmt"
	"math/rand"
	"slices"
	"sort"
)

type Player struct {
	ID     int
	TeamID int
}

func validInput(teams map[int][]int, teamSize int, tableSize int) bool {
	if teamSize%tableSize > 0 {
		return false
	}
	for teamID, members := range teams {
		if teamID < 1 {
			return false
		}

		if len(members) != teamSize {
			return false
		}
	}
	return true
}

func AssignTables(teams map[int][]int, teamSize int, tableSize int, seed int64) (map[int][]Player, error) {

	if !validInput(teams, teamSize, tableSize) {
		return nil, fmt.Errorf("Invalid input")
	}

	numberOfTeams := len(teams)
	numberOfPlayers := teamSize * numberOfTeams

	playersToAssign := make([]Player, 0, numberOfPlayers)

	teamIDs := make([]int, 0, numberOfTeams)

	for id := range teams {
		teamIDs = append(teamIDs, id)
	}

	sort.Ints(teamIDs)

	for _, teamID := range teamIDs {
		memberIDs := teams[teamID]
		teamMembers := make([]Player, 0, len(memberIDs))
		for _, id := range memberIDs {
			teamMembers = append(teamMembers, Player{
				TeamID: teamID,
				ID:     id,
			})
		}
		playersToAssign = append(playersToAssign, teamMembers...)
	}

	rnd := rand.New(rand.NewSource(seed))

	rnd.Shuffle(numberOfPlayers, func(i, j int) {
		playersToAssign[i], playersToAssign[j] = playersToAssign[j], playersToAssign[i]
	})

	numberOfTables := numberOfPlayers / tableSize

	tables := make(map[int][]Player, numberOfTables)
	for i := 0; i < numberOfTables; i++ {
		tables[i] = make([]Player, 0, tableSize)
	}

	tableIDs := make([]int, 0, numberOfTables)

	for tableID := range teams {
		tableIDs = append(tableIDs, tableID)
	}

	sort.Ints(tableIDs)

	for i := 0; i < tableSize; i++ {
		for tableID := range tableIDs {
			assignedToTable := tables[tableID]
			for i, player := range playersToAssign {
				if !slices.Contains(assignedToTable, player) {
					tables[tableID] = append(assignedToTable, player)
					playersToAssign = append(playersToAssign[:i], playersToAssign[i+1:]...)
					break
				}
			}
		}
	}

	return tables, nil
}
