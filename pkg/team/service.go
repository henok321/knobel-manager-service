package team

import (
	"context"
	"errors"

	"github.com/henok321/knobel-manager-service/gen/types"
	"github.com/henok321/knobel-manager-service/pkg/apperror"
	"github.com/henok321/knobel-manager-service/pkg/entity"
	"github.com/henok321/knobel-manager-service/pkg/game"
	"gorm.io/gorm"
)

type TeamsService interface {
	CreateTeam(ctx context.Context, gameID int, sub string, request types.TeamsRequest) (entity.Team, error)
	UpdateTeam(ctx context.Context, gameID int, sub string, teamID int, request types.TeamsRequest) (entity.Team, error)
	DeleteTeam(ctx context.Context, gameID int, sub string, teamID int) error
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

func (s *teamsService) CreateTeam(ctx context.Context, gameID int, sub string, request types.TeamsRequest) (entity.Team, error) {
	gameByID, err := s.gamesRepo.FindByID(ctx, gameID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Team{}, entity.ErrGameNotFound
		}

		return entity.Team{}, err
	}

	if !entity.IsOwner(gameByID, sub) {
		return entity.Team{}, apperror.ErrNotOwner
	}

	var playerCount int
	if request.Players != nil {
		playerCount = len(*request.Players)
	}

	if playerCount > gameByID.TeamSize {
		return entity.Team{}, apperror.ErrTeamSizeNotAllowed
	}

	players := make([]*entity.Player, playerCount)

	if request.Players != nil {
		for i, player := range *request.Players {
			players[i] = &entity.Player{Name: player.Name}
		}
	}

	team := entity.Team{
		Name:    request.Name,
		GameID:  gameID,
		Players: players,
	}

	return s.teamRepo.CreateOrUpdateTeam(ctx, &team)
}

func (s *teamsService) UpdateTeam(ctx context.Context, gameID int, sub string, teamID int, request types.TeamsRequest) (entity.Team, error) {
	gameByID, err := s.gamesRepo.FindByID(ctx, gameID)
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
			return s.teamRepo.CreateOrUpdateTeam(ctx, team)
		}
	}

	return entity.Team{}, apperror.ErrTeamNotFound
}

func (s *teamsService) DeleteTeam(ctx context.Context, gameID int, sub string, teamID int) error {
	gameByID, err := s.gamesRepo.FindByID(ctx, gameID)
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
			return s.teamRepo.DeleteTeam(ctx, teamID)
		}
	}

	return apperror.ErrTeamNotFound
}
