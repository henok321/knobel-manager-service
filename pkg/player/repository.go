package player

import (
	"github.com/henok321/knobel-manager-service/pkg/entity"
	"gorm.io/gorm"
)

type PlayersRepository interface {
	FindPlayerById(id uint) (entity.Player, error)
	CreateOrUpdatePlayer(player *entity.Player) (entity.Player, error)
	DeletePlayer(id uint) error
}

type playersRepository struct {
	db *gorm.DB
}

func NewPlayersRepository(db *gorm.DB) PlayersRepository {
	return &playersRepository{db}
}

func (r *playersRepository) FindPlayerById(id uint) (entity.Player, error) {
	player := entity.Player{}
	err := r.db.Where("id = ?", id).Preload("Team").Preload("Team.Game").Preload("Team.Game.Owners").First(&player).Error
	if err != nil {
		return entity.Player{}, err
	}

	return player, nil
}

func (r *playersRepository) CreateOrUpdatePlayer(player *entity.Player) (entity.Player, error) {
	err := r.db.Save(player).Error
	if err != nil {
		return entity.Player{}, err
	}
	return *player, nil
}

func (r *playersRepository) DeletePlayer(id uint) error {
	err := r.db.Delete(&entity.Player{}, id).Error
	if err != nil {
		return err
	}
	return nil
}