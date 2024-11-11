package player

import (
	"errors"

	"github.com/henok321/knobel-manager-service/pkg/entity"
	"github.com/henok321/knobel-manager-service/pkg/team"
	"gorm.io/gorm"
)

type PlayersService interface {
	CreatePlayer(request PlayersRequest, teamId uint, sub string) (entity.Player, error)
	UpdatePlayer(id uint, request PlayersRequest, sub string) (entity.Player, error)
	DeletePlayer(id uint, sub string) error
}

type playersService struct {
	playersRepo PlayersRepository
	teamsRepo   team.TeamsRepository
}

func NewPlayersService(playersRepo PlayersRepository, teamsRepo team.TeamsRepository) PlayersService {
	return &playersService{playersRepo: playersRepo, teamsRepo: teamsRepo}
}

func (s playersService) CreatePlayer(request PlayersRequest, teamID uint, sub string) (entity.Player, error) {
	teamById, err := s.teamsRepo.FindById(teamID)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Player{}, entity.ErrorTeamNotFound
		}
	}

	game := teamById.Game

	if !entity.IsOwner(*game, sub) {
		return entity.Player{}, entity.ErrorNotOwner
	}

	player := entity.Player{Name: request.Name, TeamID: teamID}

	createdPlayer, err := s.playersRepo.CreateOrUpdatePlayer(&player)

	if err != nil {
		return entity.Player{}, err
	}

	return createdPlayer, nil
}

func (s playersService) UpdatePlayer(id uint, request PlayersRequest, sub string) (entity.Player, error) {

	player, err := s.playersRepo.FindPlayerById(id)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Player{}, entity.ErrorPlayerNotFound
		}
		return entity.Player{}, err
	}

	game := player.Team.Game

	if !entity.IsOwner(*game, sub) {
		return entity.Player{}, entity.ErrorNotOwner
	}

	player.Name = request.Name

	updatePlayer, err := s.playersRepo.CreateOrUpdatePlayer(&player)
	if err != nil {
		return entity.Player{}, err
	}

	return updatePlayer, nil
}

func (s playersService) DeletePlayer(id uint, sub string) error {
	player, err := s.playersRepo.FindPlayerById(id)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.ErrorPlayerNotFound
		}
		return err
	}

	game := player.Team.Game

	if !entity.IsOwner(*game, sub) {
		return entity.ErrorNotOwner
	}

	err = s.playersRepo.DeletePlayer(id)
	if err != nil {
		return err
	}

	return nil
}
