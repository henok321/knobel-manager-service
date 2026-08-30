package player

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/henok321/knobel-manager-service/gen/api"
	"github.com/henok321/knobel-manager-service/pkg/apperror"
	"github.com/henok321/knobel-manager-service/pkg/entity"
	"github.com/henok321/knobel-manager-service/pkg/game"
	"github.com/henok321/knobel-manager-service/pkg/team"
)

type PlayersService struct {
	playersRepo  *PlayersRepository
	teamsRepo    *team.TeamsRepository
	gamesService *game.GamesService
}

func NewPlayersService(playersRepo *PlayersRepository, teamsRepo *team.TeamsRepository, gamesService *game.GamesService) *PlayersService {
	return &PlayersService{playersRepo: playersRepo, teamsRepo: teamsRepo, gamesService: gamesService}
}

func (s PlayersService) CreatePlayer(ctx context.Context, request api.PlayersRequest, teamID int, sub string) (entity.Player, error) {
	teamByID, err := s.teamsRepo.FindByID(ctx, teamID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Player{}, apperror.ErrTeamNotFound
		}

		return entity.Player{}, err
	}

	var created entity.Player

	err = s.gamesService.WithinSetup(ctx, teamByID.GameID, sub, func(ctx context.Context, tx *gorm.DB, _ entity.Game) error {
		player := entity.Player{Name: request.Name, TeamID: teamID}

		saved, err := NewPlayersRepository(tx).CreateOrUpdatePlayer(ctx, &player)
		if err != nil {
			return err
		}

		created = saved

		return nil
	})

	return created, err
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
	player, err := s.ownedPlayer(ctx, id, sub)
	if err != nil {
		return err
	}

	return s.gamesService.WithinSetup(ctx, player.Team.GameID, sub, func(ctx context.Context, tx *gorm.DB, _ entity.Game) error {
		return NewPlayersRepository(tx).DeletePlayer(ctx, id)
	})
}
