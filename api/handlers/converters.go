package handlers

import (
	"github.com/henok321/knobel-manager-service/gen/games"
	"github.com/henok321/knobel-manager-service/gen/players"
	"github.com/henok321/knobel-manager-service/gen/tables"
	"github.com/henok321/knobel-manager-service/gen/teams"
	"github.com/henok321/knobel-manager-service/pkg/entity"
)

func entityGameToAPIGame(gameEntity entity.Game) games.Game {
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
			teamsSlice[i] = entityTeamToAPITeam(*team)
		}
		apiGame.Teams = &teamsSlice
	}

	if len(gameEntity.Rounds) > 0 {
		rounds := make([]games.GameRound, len(gameEntity.Rounds))
		for i, round := range gameEntity.Rounds {
			rounds[i] = entityRoundToAPIRound(*round)
		}
		apiGame.Rounds = &rounds
	}

	return apiGame
}

func entityTeamToAPITeam(teamEntity entity.Team) games.Team {
	apiTeam := games.Team{
		GameID: teamEntity.GameID,
		Id:     teamEntity.ID,
		Name:   teamEntity.Name,
	}

	if len(teamEntity.Players) > 0 {
		apiPlayers := make([]games.Player, len(teamEntity.Players))
		for i, player := range teamEntity.Players {
			apiPlayers[i] = games.Player{
				Id:     player.ID,
				Name:   player.Name,
				TeamID: player.TeamID,
			}
		}
		apiTeam.Players = &apiPlayers
	}

	return apiTeam
}

func entityRoundToAPIRound(roundEntity entity.Round) games.GameRound {
	apiRound := games.GameRound{
		GameID:      roundEntity.GameID,
		Id:          roundEntity.ID,
		RoundNumber: roundEntity.RoundNumber,
		Status:      games.RoundStatus(roundEntity.Status),
	}

	if len(roundEntity.Tables) > 0 {
		tablesSlice := make([]games.Table, len(roundEntity.Tables))
		for i, table := range roundEntity.Tables {
			tablesSlice[i] = entityTableToAPITable(*table)
		}
		apiRound.Tables = &tablesSlice
	}

	return apiRound
}

func entityTableToAPITable(tablesEntity entity.GameTable) games.Table {
	apiTable := games.Table{
		Id:          tablesEntity.ID,
		RoundID:     tablesEntity.RoundID,
		TableNumber: tablesEntity.TableNumber,
	}

	if len(tablesEntity.Players) > 0 {
		apiPlayers := make([]games.Player, len(tablesEntity.Players))
		for i, player := range tablesEntity.Players {
			apiPlayers[i] = games.Player{
				Id:     player.ID,
				Name:   player.Name,
				TeamID: player.TeamID,
			}
		}
		apiTable.Players = &apiPlayers
	}

	if len(tablesEntity.Scores) > 0 {
		scores := make([]games.Score, len(tablesEntity.Scores))
		for i, score := range tablesEntity.Scores {
			scores[i] = games.Score{
				Id:       score.ID,
				PlayerID: score.PlayerID,
				Score:    score.Score,
				TableID:  score.TableID,
			}
		}
		apiTable.Scores = &scores
	}

	return apiTable
}

func entityPlayerToAPIPlayer(playersEntity entity.Player) players.Player {
	return players.Player{
		Id:     playersEntity.ID,
		Name:   playersEntity.Name,
		TeamID: playersEntity.TeamID,
	}
}

func entityTeamToTeamsAPITeam(teamEntity entity.Team) teams.Team {
	apiTeam := teams.Team{
		GameID: teamEntity.GameID,
		Id:     teamEntity.ID,
		Name:   teamEntity.Name,
	}

	if len(teamEntity.Players) > 0 {
		players := make([]teams.Player, len(teamEntity.Players))
		for i, player := range teamEntity.Players {
			players[i] = teams.Player{
				Id:     player.ID,
				Name:   player.Name,
				TeamID: player.TeamID,
			}
		}
		apiTeam.Players = &players
	}

	return apiTeam
}

func entityTableToTablesAPITable(gameEntity entity.GameTable) tables.Table {
	apiTable := tables.Table{
		Id:          gameEntity.ID,
		RoundID:     gameEntity.RoundID,
		TableNumber: gameEntity.TableNumber,
	}

	if len(gameEntity.Players) > 0 {
		apiPlayers := make([]tables.Player, len(gameEntity.Players))
		for i, player := range gameEntity.Players {
			apiPlayers[i] = tables.Player{
				Id:     player.ID,
				Name:   player.Name,
				TeamID: player.TeamID,
			}
		}
		apiTable.Players = &apiPlayers
	}

	if len(gameEntity.Scores) > 0 {
		scores := make([]tables.Score, len(gameEntity.Scores))
		for i, score := range gameEntity.Scores {
			scores[i] = tables.Score{
				Id:       score.ID,
				PlayerID: score.PlayerID,
				Score:    score.Score,
				TableID:  score.TableID,
			}
		}
		apiTable.Scores = &scores
	}

	return apiTable
}
