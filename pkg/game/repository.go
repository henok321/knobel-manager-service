package game

import (
	"github.com/henok321/knobel-manager-service/pkg/model"
	"gorm.io/gorm"
)

type GamesRepository interface {
	FindAllByOwner(sub string) ([]model.Game, error)
	FindByID(id uint) (model.Game, error)
	CreateGame(game *model.Game) (model.Game, error)
}

type gamesRepository struct {
	db *gorm.DB
}

func NewGamesRepository(db *gorm.DB) GamesRepository {
	return &gamesRepository{db}
}

func (r *gamesRepository) FindAllByOwner(sub string) ([]model.Game, error) {
	var games []model.Game

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

func (r *gamesRepository) FindByID(id uint) (model.Game, error) {
	var game model.Game

	err := r.db.
		Where("games.id = ?", id).
		Preload("Teams.Players.Scores").
		Preload("Rounds.Tables.Players").
		Preload("Rounds.Tables.Scores").
		Preload("Rounds").
		Preload("Teams").Preload("Owners").
		First(&game).Error
	if err != nil {
		return model.Game{}, err
	}

	return game, nil
}

func (r *gamesRepository) CreateGame(game *model.Game) (model.Game, error) {
	err := r.db.Create(game).Error
	if err != nil {
		return model.Game{}, err
	}

	return *game, nil
}
