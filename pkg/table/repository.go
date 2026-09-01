package table

import (
	"context"

	"gorm.io/gorm"

	"github.com/henok321/knobel-manager-service/pkg/entity"
)

type TablesRepository struct {
	db *gorm.DB
}

func NewTablesRepository(db *gorm.DB) *TablesRepository {
	return &TablesRepository{db}
}

func orderBy(column string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB { return db.Order(column) }
}

func (t *TablesRepository) FindTable(ctx context.Context, sub string, gameID, roundNumber, tableNumber int) (entity.GameTable, error) {
	tableEntity := entity.GameTable{}

	err := t.db.WithContext(ctx).
		Joins("JOIN rounds ON rounds.id = game_tables.round_id").
		Joins("JOIN game_owners ON game_owners.game_id = rounds.game_id").
		Preload("Scores", orderBy("id")).
		Preload("Players", orderBy("players.id")).
		Where("game_owners.owner_sub = ?", sub).
		Where("rounds.game_id = ?", gameID).
		Where("rounds.round_number = ?", roundNumber).
		Where("game_tables.table_number = ?", tableNumber).
		First(&tableEntity).Error
	if err != nil {
		return entity.GameTable{}, err
	}

	return tableEntity, nil
}

func (t *TablesRepository) GameStatus(ctx context.Context, gameID int) (entity.GameStatus, error) {
	var game entity.Game

	if err := t.db.WithContext(ctx).Select("status").First(&game, gameID).Error; err != nil {
		return "", err
	}

	return game.Status, nil
}

func (t *TablesRepository) UpdateTable(ctx context.Context, table *entity.GameTable) (entity.GameTable, error) {
	// Batched, not a Save per score: a partial failure would leave the audit log claiming a deliberate partial submission.
	if len(table.Scores) > 0 {
		if err := t.db.WithContext(ctx).Save(table.Scores).Error; err != nil {
			return entity.GameTable{}, err
		}
	}

	err := t.db.WithContext(ctx).Preload("Scores", orderBy("id")).Preload("Players", orderBy("players.id")).First(table, table.ID).Error
	if err != nil {
		return entity.GameTable{}, err
	}

	return *table, nil
}
