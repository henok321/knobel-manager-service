package player

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/henok321/knobel-manager-service/gen/api"
	"github.com/henok321/knobel-manager-service/pkg/apperror"
	"github.com/henok321/knobel-manager-service/pkg/entity"
	"github.com/henok321/knobel-manager-service/pkg/team"
)

type PlayersService struct {
	playersRepo *PlayersRepository
	teamsRepo   *team.TeamsRepository
}

func NewPlayersService(playersRepo *PlayersRepository, teamsRepo *team.TeamsRepository) *PlayersService {
	return &PlayersService{playersRepo: playersRepo, teamsRepo: teamsRepo}
}

func (s PlayersService) CreatePlayer(ctx context.Context, request api.PlayersRequest, teamID int, sub string) (entity.Player, error) {
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

	return s.playersRepo.CreateOrUpdatePlayer(ctx, &player)
}

func (s PlayersService) ownedPlayer(ctx context.Context, id int, sub string) (entity.Player, error) {
	player, err := s.playersRepo.FindPlayerByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Player{}, apperror.ErrPlayerNotFound
		}

		return entity.Player{}, err
	}

	if !entity.IsOwner(*player.Team.Game, sub) {
		return entity.Player{}, apperror.ErrNotOwner
	}

	return player, nil
}

func (s PlayersService) UpdatePlayer(ctx context.Context, id int, request api.PlayersRequest, sub string) (entity.Player, error) {
	player, err := s.ownedPlayer(ctx, id, sub)
	if err != nil {
		return entity.Player{}, err
	}

	player.Name = request.Name

	return s.playersRepo.CreateOrUpdatePlayer(ctx, &player)
}

func (s PlayersService) DeletePlayer(ctx context.Context, id int, sub string) error {
	if _, err := s.ownedPlayer(ctx, id, sub); err != nil {
		return err
	}

	return s.playersRepo.DeletePlayer(ctx, id)
}
