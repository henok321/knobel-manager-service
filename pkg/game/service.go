package game

import (
	"errors"

	"github.com/henok321/knobel-manager-service/pkg/entity"
	"gorm.io/gorm"
)

type GamesService interface {
	FindAllByOwner(sub string) ([]entity.Game, error)
	FindByID(id uint, sub string) (entity.Game, error)
	CreateGame(sub string, game *GameRequest) (entity.Game, error)
	UpdateGame(id uint, sub string, game GameRequest) (entity.Game, error)
	DeleteGame(id uint, sub string) error
}

type gamesService struct {
	repo GamesRepository
}

func NewGamesService(repo GamesRepository) GamesService {
	return &gamesService{repo}
}

func (s *gamesService) FindAllByOwner(sub string) ([]entity.Game, error) {
	return s.repo.FindAllByOwner(sub)
}

func (s *gamesService) FindByID(id uint, sub string) (entity.Game, error) {
	gameById, err := s.repo.FindByID(id)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Game{}, entity.ErrorGameNotFound
		}
		return entity.Game{}, err
	}

	if !entity.IsOwner(gameById, sub) {
		return entity.Game{}, entity.ErrorNotOwner
	}

	return gameById, nil
}

func (s *gamesService) CreateGame(sub string, game *GameRequest) (entity.Game, error) {
	gameModel := entity.Game{
		Name:           game.Name,
		TeamSize:       game.TeamSize,
		TableSize:      game.TableSize,
		NumberOfRounds: game.NumberOfRounds,
		Owners:         []*entity.GameOwner{{OwnerSub: sub}},
		Status:         entity.StatusSetup,
	}
	return s.repo.CreateOrUpdateGame(&gameModel)
}

func (s *gamesService) UpdateGame(id uint, sub string, game GameRequest) (entity.Game, error) {
	gameByID, err := s.repo.FindByID(id)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Game{}, entity.ErrorGameNotFound
		}
		return entity.Game{}, err
	}

	if !entity.IsOwner(gameByID, sub) {
		return entity.Game{}, entity.ErrorNotOwner
	}

	gameByID.Name = game.Name
	gameByID.TeamSize = game.TeamSize
	gameByID.TableSize = game.TableSize
	gameByID.NumberOfRounds = game.NumberOfRounds

	return s.repo.CreateOrUpdateGame(&gameByID)
}

func (s *gamesService) DeleteGame(id uint, sub string) error {
	gameByID, err := s.repo.FindByID(id)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.ErrorGameNotFound
		}
		return err
	}
	if !entity.IsOwner(gameByID, sub) {
		return entity.ErrorNotOwner
	}
	return s.repo.DeleteGame(id)
}
