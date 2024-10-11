package player

import (
	"gorm.io/gorm"
)

type PlayersRepository interface {
	Create(player *Player) error
	FindByID(id uint) (*Player, error)
	FindAll() ([]Player, error)
}

type playersRepository struct {
	db *gorm.DB
}

func NewPlayerRepository(db *gorm.DB) PlayersRepository {
	return &playersRepository{db}
}

func (r *playersRepository) Create(player *Player) error {
	return r.db.Create(player).Error
}

func (r *playersRepository) FindByID(id uint) (*Player, error) {
	var player Player
	err := r.db.First(&player, id).Error

	return &player, err
}

func (r *playersRepository) FindAll() ([]Player, error) {
	var players []Player
	err := r.db.Find(&players).Error

	return players, err
}
