package services

import (
	"knobel-manager-service/db"
	"knobel-manager-service/models"
)

type PlayersService interface {
	FindAll() ([]models.Player, error)
}

type playersService struct {
	playerRepository db.PlayerRepository
}

func NewPlayersService(playerRepository db.PlayerRepository) PlayersService {
	return &playersService{playerRepository}
}

func (s *playersService) FindAll() ([]models.Player, error) {
	players, err := s.playerRepository.FindAll()
	if err != nil {
		return nil, err
	}
	return players, nil
}
