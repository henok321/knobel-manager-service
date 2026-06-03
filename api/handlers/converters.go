package handlers

import (
	"github.com/henok321/knobel-manager-service/gen/games"
	"github.com/henok321/knobel-manager-service/gen/players"
	"github.com/henok321/knobel-manager-service/gen/scores"
	"github.com/henok321/knobel-manager-service/gen/tables"
	"github.com/henok321/knobel-manager-service/gen/teams"
	"github.com/henok321/knobel-manager-service/pkg/entity"
)

func entityPlayerToGamesPlayer(p entity.Player) games.Player {
	return games.Player{
		Id:     p.ID,
		Name:   p.Name,
		TeamID: p.TeamID,
	}
}

func entityPlayerToTeamsPlayer(p entity.Player) teams.Player {
	return teams.Player{
		Id:     p.ID,
		Name:   p.Name,
		TeamID: p.TeamID,
	}
}

func entityPlayerToPlayersPlayer(p entity.Player) players.Player {
	return players.Player{
		Id:     p.ID,
		Name:   p.Name,
		TeamID: p.TeamID,
	}
}

func entityPlayerToTablesPlayer(p entity.Player) tables.Player {
	return tables.Player{
		Id:     p.ID,
		Name:   p.Name,
		TeamID: p.TeamID,
	}
}

func entityScoreToTablesScore(s entity.Score) tables.Score {
	return tables.Score{
		Id:       s.ID,
		PlayerID: s.PlayerID,
		Score:    s.Score,
		TableID:  s.TableID,
	}
}

func entityGameToGamesGame(gameEntity entity.Game) games.Game {
	apiGame := games.Game{
		Id:             gameEntity.ID,
		Name:           gameEntity.Name,
		NumberOfRounds: gameEntity.NumberOfRounds,
		Status:         games.GameStatus(gameEntity.Status),
		TableSize:      gameEntity.TableSize,
		TeamSize:       gameEntity.TeamSize,
	}

	if len(gameEntity.Owners) > 0 {
		owners := make([]games.GameOwner, len(gameEntity.Owners))
		for i, owner := range gameEntity.Owners {
			owners[i] = games.GameOwner{
				GameID:   owner.GameID,
				OwnerSub: owner.OwnerSub,
			}
		}
		apiGame.Owners = owners
	}

	if len(gameEntity.Teams) > 0 {
		teamsSlice := make([]games.Team, len(gameEntity.Teams))
		for i, team := range gameEntity.Teams {
			teamsSlice[i] = entityTeamToGamesTeam(*team)
		}
		apiGame.Teams = &teamsSlice
	}

	if len(gameEntity.Rounds) > 0 {
		rounds := make([]games.GameRound, len(gameEntity.Rounds))
		for i, round := range gameEntity.Rounds {
			rounds[i] = entityRoundToGamesRound(*round)
		}
		apiGame.Rounds = &rounds
	}

	return apiGame
}

func entityTeamToGamesTeam(teamEntity entity.Team) games.Team {
	apiTeam := games.Team{
		GameID: teamEntity.GameID,
		Id:     teamEntity.ID,
		Name:   teamEntity.Name,
	}

	if len(teamEntity.Players) > 0 {
		apiPlayers := make([]games.Player, len(teamEntity.Players))
		for i, player := range teamEntity.Players {
			apiPlayers[i] = entityPlayerToGamesPlayer(*player)
		}
		apiTeam.Players = &apiPlayers
	}

	return apiTeam
}

func entityTeamToTeamsTeam(teamEntity entity.Team) teams.Team {
	apiTeam := teams.Team{
		GameID: teamEntity.GameID,
		Id:     teamEntity.ID,
		Name:   teamEntity.Name,
	}

	if len(teamEntity.Players) > 0 {
		players := make([]teams.Player, len(teamEntity.Players))
		for i, player := range teamEntity.Players {
			players[i] = entityPlayerToTeamsPlayer(*player)
		}
		apiTeam.Players = &players
	}

	return apiTeam
}

func entityRoundToGamesRound(roundEntity entity.Round) games.GameRound {
	return games.GameRound{
		GameID:      roundEntity.GameID,
		Id:          roundEntity.ID,
		RoundNumber: roundEntity.RoundNumber,
		Status:      games.RoundStatus(roundEntity.Status),
	}
}

func entityPlayerToScoresPlayer(p entity.Player) scores.Player {
	return scores.Player{
		Id:     p.ID,
		Name:   p.Name,
		TeamID: p.TeamID,
	}
}

func entityScoreToScoresScore(s entity.Score) scores.Score {
	return scores.Score{
		Id:       s.ID,
		PlayerID: s.PlayerID,
		Score:    s.Score,
		TableID:  s.TableID,
	}
}

func entityTableToScoresTable(tableEntity entity.GameTable) scores.Table {
	apiTable := scores.Table{
		Id:          tableEntity.ID,
		RoundID:     tableEntity.RoundID,
		TableNumber: tableEntity.TableNumber,
	}

	if len(tableEntity.Players) > 0 {
		apiPlayers := make([]scores.Player, len(tableEntity.Players))
		for i, player := range tableEntity.Players {
			apiPlayers[i] = entityPlayerToScoresPlayer(*player)
		}
		apiTable.Players = &apiPlayers
	}

	if len(tableEntity.Scores) > 0 {
		scoresSlice := make([]scores.Score, len(tableEntity.Scores))
		for i, score := range tableEntity.Scores {
			scoresSlice[i] = entityScoreToScoresScore(*score)
		}
		apiTable.Scores = &scoresSlice
	}

	return apiTable
}

func entityTableToTablesTable(tableEntity entity.GameTable) tables.Table {
	apiTable := tables.Table{
		Id:          tableEntity.ID,
		RoundID:     tableEntity.RoundID,
		TableNumber: tableEntity.TableNumber,
	}

	if len(tableEntity.Players) > 0 {
		apiPlayers := make([]tables.Player, len(tableEntity.Players))
		for i, player := range tableEntity.Players {
			apiPlayers[i] = entityPlayerToTablesPlayer(*player)
		}
		apiTable.Players = &apiPlayers
	}

	if len(tableEntity.Scores) > 0 {
		scores := make([]tables.Score, len(tableEntity.Scores))
		for i, score := range tableEntity.Scores {
			scores[i] = entityScoreToTablesScore(*score)
		}
		apiTable.Scores = &scores
	}

	return apiTable
}
