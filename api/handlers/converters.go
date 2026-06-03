package handlers

import (
	"github.com/henok321/knobel-manager-service/gen/api"
	"github.com/henok321/knobel-manager-service/pkg/entity"
)

func entityPlayerToAPIPlayer(p entity.Player) api.Player {
	return api.Player{
		Id:     p.ID,
		Name:   p.Name,
		TeamID: p.TeamID,
	}
}

func entityScoreToAPIScore(s entity.Score) api.Score {
	return api.Score{
		Id:       s.ID,
		PlayerID: s.PlayerID,
		Score:    s.Score,
		TableID:  s.TableID,
	}
}

func entityTableToAPITable(tableEntity entity.GameTable) api.Table {
	apiTable := api.Table{
		Id:          tableEntity.ID,
		RoundID:     tableEntity.RoundID,
		TableNumber: tableEntity.TableNumber,
	}

	if len(tableEntity.Players) > 0 {
		apiPlayers := make([]api.Player, len(tableEntity.Players))
		for i, player := range tableEntity.Players {
			apiPlayers[i] = entityPlayerToAPIPlayer(*player)
		}
		apiTable.Players = &apiPlayers
	}

	if len(tableEntity.Scores) > 0 {
		scoresSlice := make([]api.Score, len(tableEntity.Scores))
		for i, score := range tableEntity.Scores {
			scoresSlice[i] = entityScoreToAPIScore(*score)
		}
		apiTable.Scores = &scoresSlice
	}

	return apiTable
}

func entityTeamToAPITeam(teamEntity entity.Team) api.Team {
	apiTeam := api.Team{
		GameID: teamEntity.GameID,
		Id:     teamEntity.ID,
		Name:   teamEntity.Name,
	}

	if len(teamEntity.Players) > 0 {
		apiPlayers := make([]api.Player, len(teamEntity.Players))
		for i, player := range teamEntity.Players {
			apiPlayers[i] = entityPlayerToAPIPlayer(*player)
		}
		apiTeam.Players = &apiPlayers
	}

	return apiTeam
}

func entityRoundToAPIRound(roundEntity entity.Round) api.GameRound {
	return api.GameRound{
		GameID:      roundEntity.GameID,
		Id:          roundEntity.ID,
		RoundNumber: roundEntity.RoundNumber,
		Status:      api.RoundStatus(roundEntity.Status),
	}
}

func entityGameToAPIGame(gameEntity entity.Game) api.Game {
	apiGame := api.Game{
		Id:             gameEntity.ID,
		Name:           gameEntity.Name,
		NumberOfRounds: gameEntity.NumberOfRounds,
		Status:         api.GameStatus(gameEntity.Status),
		TableSize:      gameEntity.TableSize,
		TeamSize:       gameEntity.TeamSize,
	}

	if len(gameEntity.Owners) > 0 {
		owners := make([]api.GameOwner, len(gameEntity.Owners))
		for i, owner := range gameEntity.Owners {
			owners[i] = api.GameOwner{
				GameID:   owner.GameID,
				OwnerSub: owner.OwnerSub,
			}
		}
		apiGame.Owners = owners
	}

	if len(gameEntity.Teams) > 0 {
		teamsSlice := make([]api.Team, len(gameEntity.Teams))
		for i, team := range gameEntity.Teams {
			teamsSlice[i] = entityTeamToAPITeam(*team)
		}
		apiGame.Teams = &teamsSlice
	}

	if len(gameEntity.Rounds) > 0 {
		rounds := make([]api.GameRound, len(gameEntity.Rounds))
		for i, round := range gameEntity.Rounds {
			rounds[i] = entityRoundToAPIRound(*round)
		}
		apiGame.Rounds = &rounds
	}

	return apiGame
}
