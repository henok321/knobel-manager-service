package handlers

import (
	"github.com/henok321/knobel-manager-service/gen/games"
	"github.com/henok321/knobel-manager-service/gen/tables"
	"github.com/henok321/knobel-manager-service/gen/teams"
	"github.com/henok321/knobel-manager-service/gen/types"
	"github.com/henok321/knobel-manager-service/pkg/entity"
)

// Convert entity types to generated OpenAPI types

func entityGameToAPIGame(e entity.Game) games.Game {
	g := games.Game{
		Id:             e.ID,
		Name:           e.Name,
		NumberOfRounds: e.NumberOfRounds,
		Status:         games.GameStatus(e.Status),
		TableSize:      e.TableSize,
		TeamSize:       e.TeamSize,
	}

	// Convert owners
	if len(e.Owners) > 0 {
		owners := make([]games.GameOwner, len(e.Owners))
		for i, owner := range e.Owners {
			owners[i] = games.GameOwner{
				GameID:   owner.GameID,
				OwnerSub: owner.OwnerSub,
			}
		}
		g.Owners = owners
	}

	// Convert teams
	if len(e.Teams) > 0 {
		teamsSlice := make([]games.Team, len(e.Teams))
		for i, team := range e.Teams {
			teamsSlice[i] = entityTeamToAPITeam(*team)
		}
		g.Teams = &teamsSlice
	}

	// Convert rounds
	if len(e.Rounds) > 0 {
		rounds := make([]games.GameRound, len(e.Rounds))
		for i, round := range e.Rounds {
			rounds[i] = entityRoundToAPIRound(*round)
		}
		g.Rounds = &rounds
	}

	return g
}

func entityTeamToAPITeam(e entity.Team) games.Team {
	t := games.Team{
		GameID: e.GameID,
		Id:     e.ID,
		Name:   e.Name,
	}

	if len(e.Players) > 0 {
		players := make([]games.Player, len(e.Players))
		for i, player := range e.Players {
			players[i] = games.Player{
				Id:     player.ID,
				Name:   player.Name,
				TeamID: player.TeamID,
			}
		}
		t.Players = &players
	}

	return t
}

func entityRoundToAPIRound(e entity.Round) games.GameRound {
	r := games.GameRound{
		GameID:      e.GameID,
		Id:          e.ID,
		RoundNumber: e.RoundNumber,
		Status:      games.RoundStatus(e.Status),
	}

	if len(e.Tables) > 0 {
		tablesSlice := make([]games.Table, len(e.Tables))
		for i, table := range e.Tables {
			tablesSlice[i] = entityTableToAPITable(*table)
		}
		r.Tables = &tablesSlice
	}

	return r
}

func entityTableToAPITable(e entity.GameTable) games.Table {
	t := games.Table{
		Id:          e.ID,
		RoundID:     e.RoundID,
		TableNumber: e.TableNumber,
	}

	if len(e.Players) > 0 {
		players := make([]games.Player, len(e.Players))
		for i, player := range e.Players {
			players[i] = games.Player{
				Id:     player.ID,
				Name:   player.Name,
				TeamID: player.TeamID,
			}
		}
		t.Players = &players
	}

	if len(e.Scores) > 0 {
		scores := make([]games.Score, len(e.Scores))
		for i, score := range e.Scores {
			scores[i] = games.Score{
				Id:       score.ID,
				PlayerID: score.PlayerID,
				Score:    score.Score,
				TableID:  score.TableID,
			}
		}
		t.Scores = &scores
	}

	return t
}

// Converter functions for team handler
func entityTeamToTeamsAPITeam(e entity.Team) teams.Team {
	t := teams.Team{
		GameID: e.GameID,
		Id:     e.ID,
		Name:   e.Name,
	}

	if len(e.Players) > 0 {
		players := make([]teams.Player, len(e.Players))
		for i, player := range e.Players {
			players[i] = teams.Player{
				Id:     player.ID,
				Name:   player.Name,
				TeamID: player.TeamID,
			}
		}
		t.Players = &players
	}

	return t
}

// Converter functions for players handler (types package)
func entityPlayerToAPIPlayer(e entity.Player) types.Player {
	return types.Player{
		Id:     e.ID,
		Name:   e.Name,
		TeamID: e.TeamID,
	}
}

// Converter for types.Game (used in scores handler)
func entityGameToTypesAPIGame(e entity.Game) types.Game {
	g := types.Game{
		Id:             e.ID,
		Name:           e.Name,
		NumberOfRounds: e.NumberOfRounds,
		Status:         types.GameStatus(e.Status),
		TableSize:      e.TableSize,
		TeamSize:       e.TeamSize,
	}

	// Convert owners
	if len(e.Owners) > 0 {
		owners := make([]types.GameOwner, len(e.Owners))
		for i, owner := range e.Owners {
			owners[i] = types.GameOwner{
				GameID:   owner.GameID,
				OwnerSub: owner.OwnerSub,
			}
		}
		g.Owners = owners
	}

	// Convert teams
	if len(e.Teams) > 0 {
		teamsSlice := make([]types.Team, len(e.Teams))
		for i, team := range e.Teams {
			t := types.Team{
				GameID: team.GameID,
				Id:     team.ID,
				Name:   team.Name,
			}
			if len(team.Players) > 0 {
				players := make([]types.Player, len(team.Players))
				for j, player := range team.Players {
					players[j] = types.Player{
						Id:     player.ID,
						Name:   player.Name,
						TeamID: player.TeamID,
					}
				}
				t.Players = &players
			}
			teamsSlice[i] = t
		}
		g.Teams = &teamsSlice
	}

	// Convert rounds
	if len(e.Rounds) > 0 {
		rounds := make([]types.GameRound, len(e.Rounds))
		for i, round := range e.Rounds {
			r := types.GameRound{
				GameID:      round.GameID,
				Id:          round.ID,
				RoundNumber: round.RoundNumber,
				Status:      types.RoundStatus(round.Status),
			}
			if len(round.Tables) > 0 {
				tablesSlice := make([]types.Table, len(round.Tables))
				for j, table := range round.Tables {
					t := types.Table{
						Id:          table.ID,
						RoundID:     table.RoundID,
						TableNumber: table.TableNumber,
					}
					if len(table.Players) > 0 {
						players := make([]types.Player, len(table.Players))
						for k, player := range table.Players {
							players[k] = types.Player{
								Id:     player.ID,
								Name:   player.Name,
								TeamID: player.TeamID,
							}
						}
						t.Players = &players
					}
					if len(table.Scores) > 0 {
						scores := make([]types.Score, len(table.Scores))
						for k, score := range table.Scores {
							scores[k] = types.Score{
								Id:       score.ID,
								PlayerID: score.PlayerID,
								Score:    score.Score,
								TableID:  score.TableID,
							}
						}
						t.Scores = &scores
					}
					tablesSlice[j] = t
				}
				r.Tables = &tablesSlice
			}
			rounds[i] = r
		}
		g.Rounds = &rounds
	}

	return g
}

// Converter functions for tables handler
func entityTableToTablesAPITable(e entity.GameTable) tables.Table {
	t := tables.Table{
		Id:          e.ID,
		RoundID:     e.RoundID,
		TableNumber: e.TableNumber,
	}

	if len(e.Players) > 0 {
		players := make([]tables.Player, len(e.Players))
		for i, player := range e.Players {
			players[i] = tables.Player{
				Id:     player.ID,
				Name:   player.Name,
				TeamID: player.TeamID,
			}
		}
		t.Players = &players
	}

	if len(e.Scores) > 0 {
		scores := make([]tables.Score, len(e.Scores))
		for i, score := range e.Scores {
			scores[i] = tables.Score{
				Id:       score.ID,
				PlayerID: score.PlayerID,
				Score:    score.Score,
				TableID:  score.TableID,
			}
		}
		t.Scores = &scores
	}

	return t
}
