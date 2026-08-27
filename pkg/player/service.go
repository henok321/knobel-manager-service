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

func (s PlayersService) CreatePlayer(ctx context.Context, request api.PlayersRequest, gameID, teamID int, sub string) (entity.Player, error) {
	teamByID, err := s.teamsRepo.FindByID(ctx, teamID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Player{}, apperror.ErrTeamNotFound
		}
		return entity.Player{}, err
	}

	// The path gameID has to be the team's real game, not merely a game the caller
	// owns: otherwise any gameID in the URL is accepted, which both lies about where
	// the player lives and hides the change from the audit log.
	if teamByID.GameID != gameID {
		return entity.Player{}, apperror.ErrTeamNotFound
	}

	game := teamByID.Game

	if !entity.IsOwner(*game, sub) {
		return entity.Player{}, apperror.ErrNotOwner
	}

	player := entity.Player{Name: request.Name, TeamID: teamID}

	return s.playersRepo.CreateOrUpdatePlayer(ctx, &player)
}

func (s PlayersService) ownedPlayer(ctx context.Context, gameID, id int, sub string) (entity.Player, error) {
	player, err := s.playersRepo.FindPlayerByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Player{}, apperror.ErrPlayerNotFound
		}

		return entity.Player{}, err
	}

	// Reported as not found rather than forbidden: a mismatched gameID must not
	// reveal that the player exists somewhere else.
	if player.Team.GameID != gameID {
		return entity.Player{}, apperror.ErrPlayerNotFound
	}

	if !entity.IsOwner(*player.Team.Game, sub) {
		return entity.Player{}, apperror.ErrNotOwner
	}

	return player, nil
}

func (s PlayersService) UpdatePlayer(ctx context.Context, gameID, id int, request api.PlayersRequest, sub string) (entity.Player, error) {
	player, err := s.ownedPlayer(ctx, gameID, id, sub)
	if err != nil {
		return entity.Player{}, err
	}

	player.Name = request.Name

	return s.playersRepo.CreateOrUpdatePlayer(ctx, &player)
}

func (s PlayersService) DeletePlayer(ctx context.Context, gameID, id int, sub string) error {
	if _, err := s.ownedPlayer(ctx, gameID, id, sub); err != nil {
		return err
	}

	return s.playersRepo.DeletePlayer(ctx, id)
}
