package setup

import (
	"fmt"
	"math/rand"
	"slices"
	"sort"
)

type player struct {
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

func AssignTables(teams map[int][]int, teamSize int, tableSize int, seed int64) (map[int][]player, error) {
	for {
		if !validInput(teams, teamSize, tableSize) {
			return nil, fmt.Errorf("Invalid input")
		}

		numberOfTeams := len(teams)
		numberOfPlayers := teamSize * numberOfTeams

		playersToAssign := make([]player, 0, numberOfPlayers)

		teamIDs := make([]int, 0, numberOfTeams)

		for id := range teams {
			teamIDs = append(teamIDs, id)
		}

		sort.Ints(teamIDs)

		for _, teamID := range teamIDs {
			memberIDs := teams[teamID]
			teamMembers := make([]player, 0, len(memberIDs))
			for _, id := range memberIDs {
				teamMembers = append(teamMembers, player{
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

		tables := make(map[int][]player, numberOfTables)
		for i := 0; i < numberOfTables; i++ {
			tables[i] = make([]player, 0, tableSize)
		}

		tableIDs := make([]int, 0, numberOfTables)

		for tableID := range teams {
			tableIDs = append(tableIDs, tableID)
		}

		sort.Ints(tableIDs)

		for i := 0; i < tableSize; i++ {
			for tableID := range tableIDs {
				assignedToTable := tables[tableID]
				for i, playerToAssign := range playersToAssign {
					containsSameTeamID := slices.ContainsFunc(assignedToTable, func(p player) bool {
						return p.TeamID == playerToAssign.TeamID
					})
					if !containsSameTeamID {
						tables[tableID] = append(assignedToTable, playerToAssign)
						playersToAssign = append(playersToAssign[:i], playersToAssign[i+1:]...)
						break
					}
				}
			}
		}

		if len(playersToAssign) == 0 {
			return tables, nil
		} else {
			seed = seed + 1
		}
	}
}
