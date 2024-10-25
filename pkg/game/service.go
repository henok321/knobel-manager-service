package game

import (
	"errors"

	"github.com/henok321/knobel-manager-service/pkg/model"
	"gorm.io/gorm"
)

var ErrorGameNotFound = errors.New("game not found")
var ErrorNotOwner = errors.New("user is not the owner of the game")

type GamesService interface {
	FindAllByOwner(sub string) ([]model.Game, error)
	FindByID(id uint, sub string) (model.Game, error)
	CreateGame(game *model.Game, sub string) (model.Game, error)
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

func (s *gamesService) FindByID(id uint, sub string) (model.Game, error) {
	gameByID, err := s.repo.FindByID(id)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Game{}, ErrorGameNotFound
		}
		return model.Game{}, err
	}

	if !isOwner(gameByID, sub) {
		return model.Game{}, ErrorNotOwner
	}

	return gameByID, nil
}

func (s *gamesService) CreateGame(game *model.Game, sub string) (model.Game, error) {
	game.Owners = []*model.GameOwner{{OwnerSub: sub}}
	game.Status = model.StatusSetup
	return s.repo.CreateGame(game)
}

func isOwner(game model.Game, sub string) bool {
	for _, owner := range game.Owners {
		if owner.OwnerSub == sub {
			return true
		}
	}
	return false
}
