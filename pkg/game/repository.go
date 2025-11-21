package game

import (
	"github.com/henok321/knobel-manager-service/pkg/entity"
	"gorm.io/gorm"
)

type GamesRepository interface {
	FindAllByOwner(sub string) ([]entity.Game, error)
	FindByID(id int) (entity.Game, error)
	CreateOrUpdateGame(game *entity.Game) (entity.Game, error)
	DeleteGame(id int) error
	CreateRound(round *entity.Round) (entity.Round, error)
	CreateGameTables(gameTables []entity.GameTable) error
	ResetGameTables(gameID int) error
	WithinTransaction(fn func(txRepo GamesRepository) error) error
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
		Preload("Teams.Players").
		Preload("Owners").
		Find(&games).Error
	if err != nil {
		return nil, err
	}

	return games, nil
}

func (r *gamesRepository) FindByID(id int) (entity.Game, error) {
	var game entity.Game

	err := r.db.
		Where("games.id = ?", id).
		Preload("Teams.Players.Scores").
		Preload("Rounds.Tables.Players").
		Preload("Rounds.Tables.Scores").
		Preload("Rounds").
		Preload("Teams").
		Preload("Owners").
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

	// Reload game without associations to avoid returning stale preloaded data
	var savedGame entity.Game
	err = r.db.Preload("Owners").First(&savedGame, game.ID).Error
	if err != nil {
		return entity.Game{}, err
	}

	return savedGame, nil
}

func (r *gamesRepository) DeleteGame(id int) error {
	return r.db.Delete(&entity.Game{}, id).Error
}

func (r *gamesRepository) CreateRound(round *entity.Round) (entity.Round, error) {
	err := r.db.Save(round).Error
	if err != nil {
		return entity.Round{}, err
	}

	return *round, nil
}

func (r *gamesRepository) CreateGameTables(gameTables []entity.GameTable) error {
	return r.db.Save(gameTables).Error
}

func (r *gamesRepository) ResetGameTables(gameID int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var roundIDs []int
		if err := tx.Model(&entity.Round{}).Where("game_id = ?", gameID).Pluck("id", &roundIDs).Error; err != nil {
			return err
		}

		if len(roundIDs) > 0 {
			var tableIDs []int
			if err := tx.Model(&entity.GameTable{}).Where("round_id IN ?", roundIDs).Pluck("id", &tableIDs).Error; err != nil {
				return err
			}

			if len(tableIDs) > 0 {
				if err := tx.Where("table_id IN ?", tableIDs).Delete(&entity.Score{}).Error; err != nil {
					return err
				}

				if err := tx.Where("game_table_id IN ?", tableIDs).Delete(&entity.TablePlayer{}).Error; err != nil {
					return err
				}
			}

			if err := tx.Where("round_id IN ?", roundIDs).Delete(&entity.GameTable{}).Error; err != nil {
				return err
			}
		}

		if err := tx.Where("game_id = ?", gameID).Delete(&entity.Round{}).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *gamesRepository) WithinTransaction(fn func(txRepo GamesRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txRepo := &gamesRepository{db: tx}
		return fn(txRepo)
	})
}
