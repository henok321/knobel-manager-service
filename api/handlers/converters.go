package handlers

import (
	"github.com/henok321/knobel-manager-service/gen/types"
	"github.com/henok321/knobel-manager-service/pkg/entity"
)

func entityGameToAPIGame(e entity.Game) types.Game {
	g := types.Game{
		Id:             e.ID,
		Name:           e.Name,
		NumberOfRounds: e.NumberOfRounds,
		Status:         types.GameStatus(e.Status),
		TableSize:      e.TableSize,
		TeamSize:       e.TeamSize,
	}

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

	if len(e.Teams) > 0 {
		teamsSlice := make([]types.Team, len(e.Teams))
		for i, team := range e.Teams {
			teamsSlice[i] = entityTeamToAPITeam(*team)
		}
		g.Teams = &teamsSlice
	}

	if len(e.Rounds) > 0 {
		rounds := make([]types.GameRound, len(e.Rounds))
		for i, round := range e.Rounds {
			rounds[i] = entityRoundToAPIRound(*round)
		}
		g.Rounds = &rounds
	}

	return g
}

func entityTeamToAPITeam(e entity.Team) types.Team {
	t := types.Team{
		GameID: e.GameID,
		Id:     e.ID,
		Name:   e.Name,
	}

	if len(e.Players) > 0 {
		players := make([]types.Player, len(e.Players))
		for i, player := range e.Players {
			players[i] = types.Player{
				Id:     player.ID,
				Name:   player.Name,
				TeamID: player.TeamID,
			}
		}
		t.Players = &players
	}

	return t
}

func entityRoundToAPIRound(e entity.Round) types.GameRound {
	r := types.GameRound{
		GameID:      e.GameID,
		Id:          e.ID,
		RoundNumber: e.RoundNumber,
		Status:      types.RoundStatus(e.Status),
	}

	if len(e.Tables) > 0 {
		tablesSlice := make([]types.Table, len(e.Tables))
		for i, table := range e.Tables {
			tablesSlice[i] = entityTableToAPITable(*table)
		}
		r.Tables = &tablesSlice
	}

	return r
}

func entityTableToAPITable(e entity.GameTable) types.Table {
	t := types.Table{
		Id:          e.ID,
		RoundID:     e.RoundID,
		TableNumber: e.TableNumber,
	}

	if len(e.Players) > 0 {
		players := make([]types.Player, len(e.Players))
		for i, player := range e.Players {
			players[i] = types.Player{
				Id:     player.ID,
				Name:   player.Name,
				TeamID: player.TeamID,
			}
		}
		t.Players = &players
	}

	if len(e.Scores) > 0 {
		scores := make([]types.Score, len(e.Scores))
		for i, score := range e.Scores {
			scores[i] = types.Score{
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

func entityPlayerToAPIPlayer(e entity.Player) types.Player {
	return types.Player{
		Id:     e.ID,
		Name:   e.Name,
		TeamID: e.TeamID,
	}
}
