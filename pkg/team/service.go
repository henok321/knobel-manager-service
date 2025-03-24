package team

import (
	"errors"

	"github.com/henok321/knobel-manager-service/pkg/apperror"
	"github.com/henok321/knobel-manager-service/pkg/entity"
	"github.com/henok321/knobel-manager-service/pkg/game"
	"gorm.io/gorm"
)

type TeamsService interface {
	CreateTeam(gameID int, sub string, request TeamsRequest) (entity.Team, error)
	UpdateTeam(gameID int, sub string, teamID int, request TeamsRequest) (entity.Team, error)
	DeleteTeam(gameID int, sub string, teamID int) error
}

type teamsService struct {
	teamRepo  TeamsRepository
	gamesRepo game.GamesRepository
}

func NewTeamsService(teamRepo TeamsRepository, gameRepo game.GamesRepository) TeamsService {
	return &teamsService{
		teamRepo:  teamRepo,
		gamesRepo: gameRepo,
	}
}

func (s *teamsService) CreateTeam(gameID int, sub string, request TeamsRequest) (entity.Team, error) {
	gameByID, err := s.gamesRepo.FindByID(gameID)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Team{}, entity.ErrGameNotFound
		}
		return entity.Team{}, err
	}

	if !entity.IsOwner(gameByID, sub) {
		return entity.Team{}, apperror.ErrNotOwner
	}

	if len(request.Players) > gameByID.TeamSize {
		return entity.Team{}, apperror.ErrTeamSizeNotAllowed
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

func (s *teamsService) UpdateTeam(gameID int, sub string, teamID int, request TeamsRequest) (entity.Team, error) {
	gameByID, err := s.gamesRepo.FindByID(gameID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Team{}, entity.ErrGameNotFound
		}
		return entity.Team{}, err
	}
	if !entity.IsOwner(gameByID, sub) {
		return entity.Team{}, apperror.ErrNotOwner
	}

	for _, team := range gameByID.Teams {
		if team.ID == teamID {
			team.Name = request.Name
			return s.teamRepo.CreateOrUpdateTeam(team)
		}
	}
	return entity.Team{}, apperror.ErrTeamNotFound
}

func (s *teamsService) DeleteTeam(gameID int, sub string, teamID int) error {
	gameByID, err := s.gamesRepo.FindByID(gameID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.ErrGameNotFound
		}
		return err
	}
	if !entity.IsOwner(gameByID, sub) {
		return apperror.ErrNotOwner
	}

	for _, team := range gameByID.Teams {
		if team.ID == teamID {
			return s.teamRepo.DeleteTeam(teamID)
		}
	}

	return apperror.ErrTeamNotFound
}
