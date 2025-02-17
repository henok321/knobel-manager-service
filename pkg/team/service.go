package team

import (
	"errors"

	"github.com/henok321/knobel-manager-service/pkg/customError"

	"github.com/henok321/knobel-manager-service/pkg/entity"
	"github.com/henok321/knobel-manager-service/pkg/game"
	"gorm.io/gorm"
)

type TeamsService struct {
	teamRepo  *TeamsRepository
	gamesRepo *game.GamesRepository
}

func NewTeamsService(teamRepo *TeamsRepository, gameRepo *game.GamesRepository) *TeamsService {
	return &TeamsService{
		teamRepo:  teamRepo,
		gamesRepo: gameRepo,
	}
}

func (s *TeamsService) CreateTeam(gameID int, sub string, request TeamsRequest) (entity.Team, error) {
	gameById, err := s.gamesRepo.FindByID(gameID)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Team{}, entity.ErrorGameNotFound
		}
		return entity.Team{}, err
	}

	if !entity.IsOwner(gameById, sub) {
		return entity.Team{}, customError.NotOwner
	}

	if len(request.Players) > gameById.TeamSize {
		return entity.Team{}, customError.TeamSizeNotAllowed
	}

	players := make([]*entity.Player, len(request.Players))

	for i, player := range request.Players {
		players[i] = &entity.Player{Name: player.Name}
	}

	team := entity.Team{
		Name:    request.Name,
		GameID:  gameID,
		Players: players,
	}

	return s.teamRepo.CreateOrUpdateTeam(&team)
}

func (s *TeamsService) UpdateTeam(gameID int, sub string, teamID int, request TeamsRequest) (entity.Team, error) {
	gameById, err := s.gamesRepo.FindByID(gameID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Team{}, entity.ErrorGameNotFound
		}
		return entity.Team{}, err
	}
	if !entity.IsOwner(gameById, sub) {
		return entity.Team{}, customError.NotOwner
	}

	for _, team := range gameById.Teams {
		if team.ID == teamID {
			team.Name = request.Name
			return s.teamRepo.CreateOrUpdateTeam(team)
		}
	}
	return entity.Team{}, customError.TeamNotFound
}

func (s *TeamsService) DeleteTeam(gameID int, sub string, teamID int) error {
	gameById, err := s.gamesRepo.FindByID(gameID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.ErrorGameNotFound
		}
		return err
	}
	if !entity.IsOwner(gameById, sub) {
		return customError.NotOwner
	}

	for _, team := range gameById.Teams {
		if team.ID == teamID {
			return s.teamRepo.DeleteTeam(teamID)
		}
	}

	return customError.TeamNotFound
}
