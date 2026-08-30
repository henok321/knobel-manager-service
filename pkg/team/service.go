package team

import (
	"context"

	"github.com/henok321/knobel-manager-service/gen/api"
	"github.com/henok321/knobel-manager-service/pkg/apperror"
	"github.com/henok321/knobel-manager-service/pkg/entity"
	"github.com/henok321/knobel-manager-service/pkg/game"
)

type TeamsService struct {
	teamRepo     *TeamsRepository
	gamesService *game.GamesService
}

func NewTeamsService(teamRepo *TeamsRepository, gamesService *game.GamesService) *TeamsService {
	return &TeamsService{
		teamRepo:     teamRepo,
		gamesService: gamesService,
	}
}

func (s *TeamsService) CreateTeam(ctx context.Context, gameID int, sub string, request api.TeamsRequest) (entity.Team, error) {
	gameByID, err := s.gamesService.FindByID(ctx, gameID, sub)
	if err != nil {
		return entity.Team{}, err
	}

	var playerCount int
	if request.Players != nil {
		playerCount = len(*request.Players)
	}

	if playerCount > gameByID.TeamSize {
		return entity.Team{}, apperror.ErrTeamSizeNotAllowed
	}

	if err := game.EnsureSetupNotAssigned(gameByID); err != nil {
		return entity.Team{}, err
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

func (s *TeamsService) UpdateTeam(ctx context.Context, gameID int, sub string, teamID int, request api.TeamsRequest) (entity.Team, error) {
	gameByID, err := s.gamesService.FindByID(ctx, gameID, sub)
	if err != nil {
		return entity.Team{}, err
	}

	for _, team := range gameByID.Teams {
		if team.ID == teamID {
			team.Name = request.Name
			return s.teamRepo.CreateOrUpdateTeam(ctx, team)
		}
	}

	return entity.Team{}, apperror.ErrTeamNotFound
}

func (s *TeamsService) DeleteTeam(ctx context.Context, gameID int, sub string, teamID int) error {
	gameByID, err := s.gamesService.FindByID(ctx, gameID, sub)
	if err != nil {
		return err
	}

	for _, team := range gameByID.Teams {
		if team.ID == teamID {
			if err := game.EnsureSetupNotAssigned(gameByID); err != nil {
				return err
			}

			return s.teamRepo.DeleteTeam(ctx, teamID)
		}
	}

	return apperror.ErrTeamNotFound
}
