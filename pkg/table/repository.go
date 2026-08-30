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

func (t *TablesRepository) FindTable(ctx context.Context, sub string, gameID, roundNumber, tableNumber int) (entity.GameTable, error) {
	tableEntity := entity.GameTable{}

	err := t.db.WithContext(ctx).
		Joins("JOIN rounds ON rounds.id = game_tables.round_id").
		Joins("JOIN game_owners ON game_owners.game_id = rounds.game_id").
		Preload("Scores").
		Preload("Players").
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

func (t *TablesRepository) UpdateTable(ctx context.Context, table *entity.GameTable) (entity.GameTable, error) {
	// One batched upsert rather than a Save per score: a loop leaves earlier scores
	// committed when a later one fails, which the audit log then records as a deliberate
	// partial submission, and each Save would otherwise open its own transaction and pay
	// for its own actor round trip.
	if len(table.Scores) > 0 {
		if err := t.db.WithContext(ctx).Save(table.Scores).Error; err != nil {
			return entity.GameTable{}, err
		}
	}

	err := t.db.WithContext(ctx).Preload("Scores").Preload("Players").First(table, table.ID).Error
	if err != nil {
		return entity.GameTable{}, err
	}

	return *table, nil
}
