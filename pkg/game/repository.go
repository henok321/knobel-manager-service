package game

import (
	"context"

	"gorm.io/gorm"

	"github.com/henok321/knobel-manager-service/pkg/entity"
)

type GamesRepository struct {
	db *gorm.DB
}

func NewGamesRepository(db *gorm.DB) *GamesRepository {
	return &GamesRepository{db}
}

func (r *GamesRepository) FindAllByOwner(ctx context.Context, sub string) ([]entity.Game, error) {
	var games []entity.Game

	err := r.db.WithContext(ctx).
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

func (r *GamesRepository) FindByID(ctx context.Context, id int) (entity.Game, error) {
	var game entity.Game

	err := r.db.WithContext(ctx).
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

func (r *GamesRepository) CreateOrUpdateGame(ctx context.Context, game *entity.Game) (entity.Game, error) {
	err := r.db.WithContext(ctx).Save(game).Error
	if err != nil {
		return entity.Game{}, err
	}

	var savedGame entity.Game
	err = r.db.WithContext(ctx).Preload("Owners").First(&savedGame, game.ID).Error
	if err != nil {
		return entity.Game{}, err
	}

	return savedGame, nil
}

func (r *GamesRepository) DeleteGame(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Delete(&entity.Game{}, id).Error
}

func (r *GamesRepository) AddOwner(ctx context.Context, gameID int, sub string) error {
	return r.db.WithContext(ctx).Create(&entity.GameOwner{GameID: gameID, OwnerSub: sub}).Error
}

func (r *GamesRepository) RemoveOwner(ctx context.Context, gameID int, sub string) error {
	return r.db.WithContext(ctx).
		Where("game_id = ? AND owner_sub = ?", gameID, sub).
		Delete(&entity.GameOwner{}).Error
}

func (r *GamesRepository) CreateRound(ctx context.Context, round *entity.Round) (entity.Round, error) {
	err := r.db.WithContext(ctx).Save(round).Error
	if err != nil {
		return entity.Round{}, err
	}

	return *round, nil
}

func (r *GamesRepository) CreateGameTables(ctx context.Context, gameTables []entity.GameTable) error {
	return r.db.WithContext(ctx).Save(gameTables).Error
}

func (r *GamesRepository) ResetGameTables(ctx context.Context, gameID int) error {
	var roundIDs []int
	if err := r.db.WithContext(ctx).Model(&entity.Round{}).Where("game_id = ?", gameID).Pluck("id", &roundIDs).Error; err != nil {
		return err
	}

	if len(roundIDs) > 0 {
		var tableIDs []int
		if err := r.db.WithContext(ctx).Model(&entity.GameTable{}).Where("round_id IN ?", roundIDs).Pluck("id", &tableIDs).Error; err != nil {
			return err
		}

		if len(tableIDs) > 0 {
			if err := r.db.WithContext(ctx).Where("table_id IN ?", tableIDs).Delete(&entity.Score{}).Error; err != nil {
				return err
			}

			if err := r.db.WithContext(ctx).Where("game_table_id IN ?", tableIDs).Delete(&entity.TablePlayer{}).Error; err != nil {
				return err
			}
		}

		if err := r.db.WithContext(ctx).Where("round_id IN ?", roundIDs).Delete(&entity.GameTable{}).Error; err != nil {
			return err
		}
	}

	if err := r.db.WithContext(ctx).Where("game_id = ?", gameID).Delete(&entity.Round{}).Error; err != nil {
		return err
	}

	return nil
}

func (r *GamesRepository) WithinTransaction(ctx context.Context, operation func(ctx context.Context, txRepo *GamesRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := &GamesRepository{db: tx}
		return operation(ctx, txRepo)
	})
}
