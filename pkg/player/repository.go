package player

import (
	"context"

	"github.com/henok321/knobel-manager-service/pkg/entity"
	"gorm.io/gorm"
)

type PlayersRepository interface {
	FindPlayerByID(ctx context.Context, id int) (entity.Player, error)
	CreateOrUpdatePlayer(ctx context.Context, player *entity.Player) (entity.Player, error)
	DeletePlayer(ctx context.Context, id int) error
}

type playersRepository struct {
	db *gorm.DB
}

func NewPlayersRepository(db *gorm.DB) PlayersRepository {
	return &playersRepository{db}
}

func (r *playersRepository) FindPlayerByID(ctx context.Context, id int) (entity.Player, error) {
	player := entity.Player{}

	err := r.db.WithContext(ctx).Where("id = ?", id).Preload("Team").Preload("Team.Game").Preload("Team.Game.Owners").First(&player).Error
	if err != nil {
		return entity.Player{}, err
	}

	return player, nil
}

func (r *playersRepository) CreateOrUpdatePlayer(ctx context.Context, player *entity.Player) (entity.Player, error) {
	err := r.db.WithContext(ctx).Save(player).Error
	if err != nil {
		return entity.Player{}, err
	}

	return *player, nil
}

func (r *playersRepository) DeletePlayer(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Delete(&entity.Player{}, id).Error
}
