package player

import "github.com/henok321/knobel-manager-service/pkg/model"

type PlayersService interface {
	FindByGame(gameID uint, ownerID string) ([]model.Player, error)
	FindByTeam(gameID uint, teamID uint, ownerID string) ([]model.Player, error)
}

type playersService struct {
	playerRepository PlayersRepository
}

func NewPlayersService(playerRepository PlayersRepository) PlayersService {
	return &playersService{playerRepository}
}

func (s *playersService) FindByGame(teamID uint, ownerID string) ([]model.Player, error) {
	return s.playerRepository.FindByGame(teamID, ownerID)
}

func (s *playersService) FindByTeam(gameID uint, teamID uint, ownerID string) ([]model.Player, error) {
	return s.playerRepository.FindByTeam(gameID, teamID, ownerID)

}
