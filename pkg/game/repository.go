package game

import (
	"github.com/henok321/knobel-manager-service/pkg/model"
	"gorm.io/gorm"
)

type GamesRepository interface {
	FindAllByOwner(sub string) ([]model.Game, error)
}

type gamesRepository struct {
	db *gorm.DB
}

func NewGamesRepository(db *gorm.DB) GamesRepository {
	return &gamesRepository{db}
}

func (r *gamesRepository) FindAllByOwner(sub string) ([]model.Game, error) {
	var games []model.Game
	err := r.db.Where("owner_sub = ?", sub).Find(&games).Error
	if err != nil {
		return nil, err
	}

	return games, nil
}
