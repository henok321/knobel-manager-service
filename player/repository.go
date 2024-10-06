package player

import (
	"gorm.io/gorm"
)

type PlayerRepository interface {
	Create(player *Player) error
	FindByID(id uint) (*Player, error)
	FindAll() ([]Player, error)
}

type playerRepository struct {
	db *gorm.DB
}

func NewPlayerRepository(db *gorm.DB) PlayerRepository {
	return &playerRepository{db}
}

func (r *playerRepository) Create(player *Player) error {
	return r.db.Create(player).Error
}

func (r *playerRepository) FindByID(id uint) (*Player, error) {
	var player Player
	err := r.db.First(&player, id).Error
	return &player, err
}

func (r *playerRepository) FindAll() ([]Player, error) {
	var players []Player
	err := r.db.Find(&players).Error
	return players, err
}
