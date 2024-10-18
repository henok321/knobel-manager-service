package game

import (
	"github.com/henok321/knobel-manager-service/pkg/model"
	"gorm.io/gorm"
)

type GamesRepository interface {
	FindAll() ([]model.Game, error)
	FindById(id uint) (*model.Game, error)
	FindByOwner(sub string) ([]model.Game, error)
	Create(game *model.Game) error
	Update(game *model.Game) error
	Delete(id uint) error
}

type gamesRepository struct {
	db *gorm.DB
}

func NewGamesRepository(db *gorm.DB) GamesRepository {
	return &gamesRepository{db}
}

func (r *gamesRepository) FindAll() ([]model.Game, error) {
	var games []model.Game
	err := r.db.Preload("Owners").Preload("Teams").Find(&games).Error

	return games, err
}

func (r *gamesRepository) FindById(id uint) (*model.Game, error) {
	var game *model.Game
	err := r.db.Joins("JOIN game_owners ON game_owners.game_id = games.id").
		Joins("JOIN owners ON owners.id = game_owners.owner_id").
		Where("games.id = ?", id).
		Preload("Owners").
		First(&game).Error

	return game, err
}

func (r *gamesRepository) FindByOwner(sub string) ([]model.Game, error) {
	var games []model.Game
	err := r.db.Joins("JOIN game_owners ON game_owners.game_id = games.id").
		Joins("JOIN owners ON owners.id = game_owners.owner_id").
		Where("owners.sub = ?", sub).
		Preload("Owners").
		Preload("Teams").
		Find(&games).Error

	return games, err
}

func (r *gamesRepository) Create(game *model.Game) error {
	return r.db.Create(game).Error
}

func (r *gamesRepository) Update(game *model.Game) error {
	return r.db.Save(game).Error
}

func (r *gamesRepository) Delete(id uint) error {
	return r.db.Delete(&model.Game{}, id).Error
}
