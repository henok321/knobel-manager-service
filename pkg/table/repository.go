package table

import (
	"context"

	"github.com/henok321/knobel-manager-service/pkg/entity"
	"gorm.io/gorm"
)

type TablesRepository interface {
	FindTable(ctx context.Context, sub string, gameID, roundNumber, tableNumber int) (entity.GameTable, error)
	UpdateTable(ctx context.Context, table *entity.GameTable) (entity.GameTable, error)
}

type tablesRepository struct {
	db *gorm.DB
}

func NewTablesRepository(db *gorm.DB) TablesRepository {
	return &tablesRepository{db}
}

func (t *tablesRepository) FindTable(ctx context.Context, sub string, gameID, roundNumber, tableNumber int) (entity.GameTable, error) {
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

func (t *tablesRepository) UpdateTable(ctx context.Context, table *entity.GameTable) (entity.GameTable, error) {
	for _, score := range table.Scores {
		err := t.db.WithContext(ctx).Save(score).Error
		if err != nil {
			return entity.GameTable{}, err
		}
	}

	err := t.db.WithContext(ctx).Preload("Scores").Preload("Players").First(table, table.ID).Error
	if err != nil {
		return entity.GameTable{}, err
	}

	return *table, nil
}
