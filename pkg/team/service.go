package team

import (
	"errors"

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
	gameById, err := s.gamesRepo.FindByID(gameID)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Team{}, entity.ErrorGameNotFound
		}
		return entity.Team{}, err
	}

	if !entity.IsOwner(gameById, sub) {
		return entity.Team{}, entity.ErrorNotOwner
	}

	team := entity.Team{
		Name:   request.Name,
		GameID: gameID,
	}

	return s.teamRepo.CreateOrUpdateTeam(&team)
}

func (s *teamsService) UpdateTeam(gameID int, sub string, teamID int, request TeamsRequest) (entity.Team, error) {
	gameById, err := s.gamesRepo.FindByID(gameID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Team{}, entity.ErrorGameNotFound
		}
		return entity.Team{}, err
	}
	if !entity.IsOwner(gameById, sub) {
		return entity.Team{}, entity.ErrorNotOwner
	}

	for _, team := range gameById.Teams {
		if team.ID == teamID {
			team.Name = request.Name
			return s.teamRepo.CreateOrUpdateTeam(team)
		}
	}
	return entity.Team{}, entity.ErrorTeamNotFound
}

func (s *teamsService) DeleteTeam(gameID int, sub string, teamID int) error {
	gameById, err := s.gamesRepo.FindByID(gameID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.ErrorGameNotFound
		}
		return err
	}
	if !entity.IsOwner(gameById, sub) {
		return entity.ErrorNotOwner
	}

	for _, team := range gameById.Teams {
		if team.ID == teamID {
			return s.teamRepo.DeleteTeam(teamID)
		}
	}

	return entity.ErrorTeamNotFound
}
