package player

import (
	"errors"

	"github.com/henok321/knobel-manager-service/pkg/apperror"
	"github.com/henok321/knobel-manager-service/pkg/entity"
	"github.com/henok321/knobel-manager-service/pkg/team"
	"gorm.io/gorm"
)

type PlayersService interface {
	CreatePlayer(request PlayersRequest, teamID int, sub string) (entity.Player, error)
	UpdatePlayer(id int, request PlayersRequest, sub string) (entity.Player, error)
	DeletePlayer(id int, sub string) error
}

type playersService struct {
	playersRepo PlayersRepository
	teamsRepo   team.TeamsRepository
}

func NewPlayersService(playersRepo PlayersRepository, teamsRepo team.TeamsRepository) PlayersService {
	return &playersService{playersRepo: playersRepo, teamsRepo: teamsRepo}
}

func (s playersService) CreatePlayer(request PlayersRequest, teamID int, sub string) (entity.Player, error) {
	teamByID, err := s.teamsRepo.FindByID(teamID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Player{}, apperror.ErrTeamNotFound
		}
	}

	game := teamByID.Game

	if !entity.IsOwner(*game, sub) {
		return entity.Player{}, apperror.ErrNotOwner
	}

	player := entity.Player{Name: request.Name, TeamID: teamID}

	createdPlayer, err := s.playersRepo.CreateOrUpdatePlayer(&player)
	if err != nil {
		return entity.Player{}, err
	}

	return createdPlayer, nil
}

func (s playersService) UpdatePlayer(id int, request PlayersRequest, sub string) (entity.Player, error) {
	player, err := s.playersRepo.FindPlayerByID(id)
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

	updatePlayer, err := s.playersRepo.CreateOrUpdatePlayer(&player)
	if err != nil {
		return entity.Player{}, err
	}

	return updatePlayer, nil
}

func (s playersService) DeletePlayer(id int, sub string) error {
	player, err := s.playersRepo.FindPlayerByID(id)
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

	err = s.playersRepo.DeletePlayer(id)
	if err != nil {
		return err
	}

	return nil
}
