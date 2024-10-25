package game

import (
	"github.com/henok321/knobel-manager-service/pkg/entity"
	"gorm.io/gorm"
)

type GamesRepository interface {
	FindAllByOwner(sub string) ([]entity.Game, error)
	FindByID(id uint) (entity.Game, error)
	CreateOrUpdateGame(game *entity.Game) (entity.Game, error)
	DeleteGame(id uint) error
}

type gamesRepository struct {
	db *gorm.DB
}

func NewGamesRepository(db *gorm.DB) GamesRepository {
	return &gamesRepository{db}
}

func (r *gamesRepository) FindAllByOwner(sub string) ([]entity.Game, error) {
	var games []entity.Game

	err := r.db.
		Joins("JOIN game_owners ON game_owners.game_id = games.id").Where("game_owners.owner_sub = ?", sub).
		Preload("Teams.Players.Scores").
		Preload("Rounds.Tables.Players").
		Preload("Rounds.Tables.Scores").
		Preload("Rounds").
		Preload("Teams").
		Preload("Teams").
		Preload("Owners").
		Find(&games).Error
	if err != nil {
		return nil, err
	}

	return games, nil
}

func (r *gamesRepository) FindByID(id uint) (entity.Game, error) {
	var game entity.Game

	err := r.db.
		Where("games.id = ?", id).
		Preload("Teams.Players.Scores").
		Preload("Rounds.Tables.Players").
		Preload("Rounds.Tables.Scores").
		Preload("Rounds").
		Preload("Teams").Preload("Owners").
		First(&game).Error
	if err != nil {
		return entity.Game{}, err
	}

	return game, nil
}

func (r *gamesRepository) CreateOrUpdateGame(game *entity.Game) (entity.Game, error) {
	err := r.db.Save(game).Error
	if err != nil {
		return entity.Game{}, err
	}

	return *game, nil
}

func (r *gamesRepository) DeleteGame(id uint) error {
	err := r.db.Delete(&entity.Game{}, id).Error
	if err != nil {
		return err
	}

	return nil
}
