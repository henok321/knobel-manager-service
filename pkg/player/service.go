package player

import (
	"errors"

	"github.com/henok321/knobel-manager-service/pkg/customError"

	"github.com/henok321/knobel-manager-service/pkg/entity"
	"github.com/henok321/knobel-manager-service/pkg/team"
	"gorm.io/gorm"
)

type PlayersService struct {
	playersRepo *PlayersRepository
	teamsRepo   *team.TeamsRepository
}

func NewPlayersService(playersRepo *PlayersRepository, teamsRepo *team.TeamsRepository) *PlayersService {
	return &PlayersService{playersRepo: playersRepo, teamsRepo: teamsRepo}
}

func (s PlayersService) CreatePlayer(request PlayersRequest, teamID int, sub string) (entity.Player, error) {
	teamById, err := s.teamsRepo.FindById(teamID)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Player{}, customError.TeamNotFound
		}
	}

	game := teamById.Game

	if !entity.IsOwner(*game, sub) {
		return entity.Player{}, customError.NotOwner
	}

	player := entity.Player{Name: request.Name, TeamID: teamID}

	createdPlayer, err := s.playersRepo.CreateOrUpdatePlayer(&player)

	if err != nil {
		return entity.Player{}, err
	}

	return createdPlayer, nil
}

func (s PlayersService) UpdatePlayer(id int, request PlayersRequest, sub string) (entity.Player, error) {

	player, err := s.playersRepo.FindPlayerById(id)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Player{}, customError.PlayerNotFound
		}
		return entity.Player{}, err
	}

	game := player.Team.Game

	if !entity.IsOwner(*game, sub) {
		return entity.Player{}, customError.NotOwner
	}

	player.Name = request.Name

	updatePlayer, err := s.playersRepo.CreateOrUpdatePlayer(&player)
	if err != nil {
		return entity.Player{}, err
	}

	return updatePlayer, nil
}

func (s PlayersService) DeletePlayer(id int, sub string) error {
	player, err := s.playersRepo.FindPlayerById(id)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return customError.PlayerNotFound
		}
		return err
	}

	game := player.Team.Game

	if !entity.IsOwner(*game, sub) {
		return customError.NotOwner
	}

	err = s.playersRepo.DeletePlayer(id)
	if err != nil {
		return err
	}

	return nil
}
