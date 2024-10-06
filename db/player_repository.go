package db

import (
	"gorm.io/gorm"
	"knobel-manager-service/models"
)

type PlayerRepository interface {
	Create(player *models.Player) error
	FindByID(id uint) (*models.Player, error)
	FindAll() ([]models.Player, error)
}

type playerRepository struct {
	db *gorm.DB
}

func NewPlayerRepository(db *gorm.DB) PlayerRepository {
	return &playerRepository{db}
}

func (r *playerRepository) Create(player *models.Player) error {
	return r.db.Create(player).Error
}

func (r *playerRepository) FindByID(id uint) (*models.Player, error) {
	var player models.Player
	err := r.db.First(&player, id).Error
	return &player, err
}

func (r *playerRepository) FindAll() ([]models.Player, error) {
	var players []models.Player
	err := r.db.Find(&players).Error
	return players, err
}
