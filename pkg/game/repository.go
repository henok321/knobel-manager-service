package game

import "gorm.io/gorm"

type GamesRepository interface {
	FindAll() ([]Game, error)
	FindById(id uint) (*Game, error)
	FindByOwner(sub string) ([]Game, error)
	Create(game *Game) error
	Update(game *Game) error
	Delete(id uint) error
}

type gamesRepository struct {
	db *gorm.DB
}

func NewGamesRepository(db *gorm.DB) GamesRepository {
	return &gamesRepository{db}
}

func (r *gamesRepository) FindAll() ([]Game, error) {
	var games []Game
	err := r.db.Find(&games).Error

	return games, err
}

func (r *gamesRepository) FindById(id uint) (*Game, error) {
	var game *Game
	err := r.db.Joins("JOIN game_owners ON game_owners.game_id = games.id").
		Joins("JOIN owners ON owners.id = game_owners.owner_id").
		Where("games.id = ?", id).
		Preload("Owners").
		First(&game).Error

	return game, err
}

func (r *gamesRepository) FindByOwner(sub string) ([]Game, error) {
	var games []Game
	err := r.db.Joins("JOIN game_owners ON game_owners.game_id = games.id").
		Joins("JOIN owners ON owners.id = game_owners.owner_id").
		Where("owners.sub = ?", sub).
		Preload("Owners").
		Find(&games).Error

	return games, err
}

func (r *gamesRepository) Create(game *Game) error {
	return r.db.Create(game).Error
}

func (r *gamesRepository) Update(game *Game) error {
	return r.db.Save(game).Error
}

func (r *gamesRepository) Delete(id uint) error {
	return r.db.Delete(&Game{}, id).Error
}
