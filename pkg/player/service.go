package player

import (
	"context"
	"errors"

	"github.com/henok321/knobel-manager-service/gen/players"
	"github.com/henok321/knobel-manager-service/pkg/apperror"
	"github.com/henok321/knobel-manager-service/pkg/entity"
	"github.com/henok321/knobel-manager-service/pkg/team"
	"gorm.io/gorm"
)

type PlayersService interface {
	CreatePlayer(ctx context.Context, request players.PlayersRequest, teamID int, sub string) (entity.Player, error)
	UpdatePlayer(ctx context.Context, id int, request players.PlayersRequest, sub string) (entity.Player, error)
	DeletePlayer(ctx context.Context, id int, sub string) error
}

type playersService struct {
	playersRepo PlayersRepository
	teamsRepo   team.TeamsRepository
}

func NewPlayersService(playersRepo PlayersRepository, teamsRepo team.TeamsRepository) PlayersService {
	return &playersService{playersRepo: playersRepo, teamsRepo: teamsRepo}
}

func (s playersService) CreatePlayer(ctx context.Context, request players.PlayersRequest, teamID int, sub string) (entity.Player, error) {
	teamByID, err := s.teamsRepo.FindByID(ctx, teamID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Player{}, apperror.ErrTeamNotFound
		}
		return entity.Player{}, err
	}

	game := teamByID.Game

	if !entity.IsOwner(*game, sub) {
		return entity.Player{}, apperror.ErrNotOwner
	}

	player := entity.Player{Name: request.Name, TeamID: teamID}

	createdPlayer, err := s.playersRepo.CreateOrUpdatePlayer(ctx, &player)
	if err != nil {
		return entity.Player{}, err
	}

	return createdPlayer, nil
}

func (s playersService) UpdatePlayer(ctx context.Context, id int, request players.PlayersRequest, sub string) (entity.Player, error) {
	player, err := s.playersRepo.FindPlayerByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Player{}, apperror.ErrPlayerNotFound
		}

		return entity.Player{}, err
	}

	game := player.Team.Game

	if !entity.IsOwner(*game, sub) {
		return entity.Player{}, apperror.ErrNotOwner
	}

	player.Name = request.Name

	updatePlayer, err := s.playersRepo.CreateOrUpdatePlayer(ctx, &player)
	if err != nil {
		return entity.Player{}, err
	}

	return updatePlayer, nil
}

func (s playersService) DeletePlayer(ctx context.Context, id int, sub string) error {
	player, err := s.playersRepo.FindPlayerByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.ErrPlayerNotFound
		}

		return err
	}

	game := player.Team.Game

	if !entity.IsOwner(*game, sub) {
		return apperror.ErrNotOwner
	}

	return s.playersRepo.DeletePlayer(ctx, id)
}
