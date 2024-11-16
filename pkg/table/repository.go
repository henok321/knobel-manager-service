package table

import (
	"github.com/henok321/knobel-manager-service/pkg/entity"
	"gorm.io/gorm"
)

type TablesRepository interface {
	FindTables(gameID uint) []entity.GameTable
	FindTable(tableID uint) entity.GameTable
}

type tablesRepository struct {
	db *gorm.DB
}

func NewTablesRepository(db *gorm.DB) TablesRepository {
	return tablesRepository{db: db}
}

func (t tablesRepository) FindTables(gameID uint) []entity.GameTable {
	//TODO implement me
	panic("implement me")
}

func (t tablesRepository) FindTable(tableID uint) entity.GameTable {
	//TODO implement me
	panic("implement me")
}
