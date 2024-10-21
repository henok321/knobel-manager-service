package game

import "github.com/henok321/knobel-manager-service/pkg/model"

type GamesService interface {
	FindAllByOwner(sub string) ([]model.Game, error)
}

type gamesService struct {
	repo GamesRepository
}

func NewGamesService(repo GamesRepository) GamesService {
	return &gamesService{repo}
}

func (s *gamesService) FindAllByOwner(sub string) ([]model.Game, error) {
	return s.repo.FindAllByOwner(sub)
}
