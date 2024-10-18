package player

import (
	"github.com/henok321/knobel-manager-service/pkg/model"
	"gorm.io/gorm"
)

type PlayersRepository interface {
	FindByGame(gameID uint, ownerID string) ([]model.Player, error)
	FindByTeam(gameID uint, teamID uint, ownerID string) ([]model.Player, error)
}

type playersRepository struct {
	db *gorm.DB
}

func NewPlayerRepository(db *gorm.DB) PlayersRepository {
	return &playersRepository{db}
}

func (r *playersRepository) FindByGame(gameID uint, ownerID string) ([]model.Player, error) {
	var players []model.Player
	err := r.db.Joins("JOIN teams ON teams.id = players.team_id").
		Joins("JOIN games ON games.id = teams.game_id").
		Joins("JOIN game_owners ON game_owners.game_id = games.id").
		Joins("JOIN owners ON owners.id = game_owners.owner_id").
		Where("games.id = ? AND owners.sub = ?", gameID, ownerID).
		Find(&players).Error

	return players, err
}

func (r *playersRepository) FindByTeam(gameID uint, teamID uint, ownerID string) ([]model.Player, error) {
	var players []model.Player

	err := r.db.Joins("JOIN teams ON teams.id = players.team_id").
		Joins("JOIN games ON games.id = teams.game_id").
		Joins("JOIN game_owners ON game_owners.game_id = games.id").
		Joins("JOIN owners ON owners.id = game_owners.owner_id").
		Where("games.id = ? AND teams.id = ? AND owners.sub = ?", gameID, teamID, ownerID).
		Find(&players).Error

	return players, err
}
