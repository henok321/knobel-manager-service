package setup

import (
	"fmt"
	"maps"
	"math/rand"
	"slices"
)

type Player struct {
	ID     int
	TeamID int
}

func IsAssignable(teams map[int][]int, teamSize, tableSize int) bool {
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
	numberOfPlayers := len(teams) * teamSize
	numberOfTables := numberOfPlayers / tableSize

	return numberOfTables >= teamSize
}

type TeamSetup struct {
	Teams     map[int][]int
	TeamSize  int
	TableSize int
}

func AssignTables(teamSetup TeamSetup, seed int64) (map[int][]Player, error) {
	if !IsAssignable(teamSetup.Teams, teamSetup.TeamSize, teamSetup.TableSize) {
		return nil, fmt.Errorf("invalid setup: teams=%d teamSize=%d tableSize=%d", len(teamSetup.Teams), teamSetup.TeamSize, teamSetup.TableSize)
	}

	numberOfPlayers := teamSetup.TeamSize * len(teamSetup.Teams)
	numberOfTables := numberOfPlayers / teamSetup.TableSize

	allPlayers := make([]Player, 0, numberOfPlayers)

	for _, teamID := range slices.Sorted(maps.Keys(teamSetup.Teams)) {
		for _, id := range teamSetup.Teams[teamID] {
			allPlayers = append(allPlayers, Player{TeamID: teamID, ID: id})
		}
	}

	for {
		playersToAssign := slices.Clone(allPlayers)

		rnd := rand.New(rand.NewSource(seed)) //nolint:gosec // G404: deterministic seeded shuffle for table assignment, not security-sensitive

		rnd.Shuffle(numberOfPlayers, func(i, j int) {
			playersToAssign[i], playersToAssign[j] = playersToAssign[j], playersToAssign[i]
		})

		tables := make(map[int][]Player, numberOfTables)
		for i := range numberOfTables {
			tables[i] = make([]Player, 0, teamSetup.TableSize)
		}

		for range teamSetup.TableSize {
			for tableID := range numberOfTables {
				assignedToTable := tables[tableID]
				for i, playerToAssign := range playersToAssign {
					containsSameTeamID := slices.ContainsFunc(assignedToTable, func(p Player) bool {
						return p.TeamID == playerToAssign.TeamID
					})
					if !containsSameTeamID {
						tables[tableID] = append(assignedToTable, playerToAssign)
						playersToAssign = slices.Delete(playersToAssign, i, i+1)

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
