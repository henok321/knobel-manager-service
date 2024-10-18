package game

import "github.com/henok321/knobel-manager-service/pkg/model"

type GamesService interface {
	FindAllByOwner(sub string) ([]model.Game, error)
	FindByID(id uint) (*model.Game, error)
	Create(game *model.Game) error
	Update(game *model.Game) error
	Delete(id uint) error
}

type gamesService struct {
	repo GamesRepository
}

func NewGamesService(repo GamesRepository) GamesService {
	return &gamesService{repo}
}

func (s *gamesService) FindAllByOwner(sub string) ([]model.Game, error) {
	return s.repo.FindByOwner(sub)
}

func (s *gamesService) FindByID(id uint) (*model.Game, error) {
	return s.repo.FindById(id)

}

func (s *gamesService) Create(game *model.Game) error {
	return s.repo.Create(game)
}

func (s *gamesService) Update(game *model.Game) error {
	return s.repo.Update(game)
}

func (s *gamesService) Delete(id uint) error {
	return s.repo.Delete(id)
}
