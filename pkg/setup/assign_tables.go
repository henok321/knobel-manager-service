package setup

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

type TeamsPlayersMapping map[int][]Player

type TeamSetup struct {
	Teams     map[int][]int
	TeamSize  int
	TableSize int
}

func AssignTables(teamSetup TeamSetup, seed int64) (TeamsPlayersMapping, error) {
	for {
		if !validInput(teamSetup.Teams, teamSetup.TeamSize, teamSetup.TableSize) {
			return nil, fmt.Errorf("invalid input")
		}

		numberOfTeams := len(teamSetup.Teams)
		numberOfPlayers := teamSetup.TeamSize * numberOfTeams

		playersToAssign := make([]Player, 0, numberOfPlayers)

		teamIDs := make([]int, 0, numberOfTeams)

		for id := range teamSetup.Teams {
			teamIDs = append(teamIDs, id)
		}

		sort.Ints(teamIDs)

		for _, teamID := range teamIDs {
			memberIDs := teamSetup.Teams[teamID]
			teamMembers := make([]Player, 0, len(memberIDs))
			for _, id := range memberIDs {
				teamMembers = append(teamMembers, Player{
					TeamID: teamID,
					ID:     id,
				})
			}
			playersToAssign = append(playersToAssign, teamMembers...)
		}

		rnd := rand.New(rand.NewSource(seed)) //nolint:gosec

		rnd.Shuffle(numberOfPlayers, func(i, j int) {
			playersToAssign[i], playersToAssign[j] = playersToAssign[j], playersToAssign[i]
		})

		numberOfTables := numberOfPlayers / teamSetup.TableSize

		tables := make(map[int][]Player, numberOfTables)
		for i := 0; i < numberOfTables; i++ {
			tables[i] = make([]Player, 0, teamSetup.TableSize)
		}

		tableIDs := make([]int, 0, numberOfTables)

		for tableID := range teamSetup.Teams {
			tableIDs = append(tableIDs, tableID)
		}

		sort.Ints(tableIDs)

		for i := 0; i < teamSetup.TableSize; i++ {
			for tableID := range tableIDs {
				assignedToTable := tables[tableID]
				for i, playerToAssign := range playersToAssign {
					containsSameTeamID := slices.ContainsFunc(assignedToTable, func(p Player) bool {
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
		}
		seed++
	}
}
