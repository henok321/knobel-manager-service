package handlers

import (
	"github.com/henok321/knobel-manager-service/pkg/entity"
)

type TablesHandler interface {
	GetTables(gameID uint) []entity.GameTable
	GetTable(tableID uint) entity.GameTable
}

type tablesHandler struct {
}

func NewTablesHandler() TablesHandler {
	return &tablesHandler{}
}

func (t tablesHandler) GetTables(gameID uint) []entity.GameTable {
	//TODO implement me
	panic("implement me")
}

func (t tablesHandler) GetTable(tableID uint) entity.GameTable {
	//TODO implement me
	panic("implement me")
}
