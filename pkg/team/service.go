package team

import (
	"context"
	"errors"

	"gorm.io/gorm"

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
	var created entity.Team

	err := s.gamesService.WithinSetup(ctx, gameID, sub, func(ctx context.Context, tx *gorm.DB, gameByID entity.Game) error {
		var playerCount int
		if request.Players != nil {
			playerCount = len(*request.Players)
		}

		if playerCount > gameByID.TeamSize {
			return apperror.ErrTeamSizeNotAllowed
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

		saved, err := NewTeamsRepository(tx).CreateOrUpdateTeam(ctx, &team)
		if err != nil {
			return err
		}

		created = saved

		return nil
	})

	return created, err
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
	return s.gamesService.WithinSetup(ctx, gameID, sub, func(ctx context.Context, tx *gorm.DB, _ entity.Game) error {
		txRepo := NewTeamsRepository(tx)

		teamByID, err := txRepo.FindByID(ctx, teamID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperror.ErrTeamNotFound
			}

			return err
		}

		if teamByID.GameID != gameID {
			return apperror.ErrTeamNotFound
		}

		return txRepo.DeleteTeam(ctx, teamID)
	})
}
