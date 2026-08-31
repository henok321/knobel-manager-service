package player

import (
	"context"

	"gorm.io/gorm"

	"github.com/henok321/knobel-manager-service/pkg/entity"
)

type PlayersRepository struct {
	db *gorm.DB
}

func NewPlayersRepository(db *gorm.DB) *PlayersRepository {
	return &PlayersRepository{db}
}

func (r *PlayersRepository) FindPlayerByID(ctx context.Context, id int) (entity.Player, error) {
	player := entity.Player{}

	err := r.db.WithContext(ctx).Where("id = ?", id).Preload("Team").Preload("Team.Game").Preload("Team.Game.Owners").First(&player).Error
	if err != nil {
		return entity.Player{}, err
	}

	return player, nil
}

func (r *PlayersRepository) CreatePlayer(ctx context.Context, player *entity.Player) (entity.Player, error) {
	err := r.db.WithContext(ctx).Save(player).Error
	if err != nil {
		return entity.Player{}, err
	}

	return *player, nil
}

func (r *PlayersRepository) UpdatePlayerName(ctx context.Context, player *entity.Player, name string) (entity.Player, error) {
	if err := r.db.WithContext(ctx).Model(&entity.Player{ID: player.ID}).Update("player_name", name).Error; err != nil {
		return entity.Player{}, err
	}

	player.Name = name

	return *player, nil
}

func (r *PlayersRepository) DeletePlayer(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Delete(&entity.Player{}, id).Error
}
