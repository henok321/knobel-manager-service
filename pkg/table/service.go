package table

import "github.com/henok321/knobel-manager-service/pkg/entity"

type TablesService interface {
	FindTables(gameID uint, sub string) []entity.GameTable
	FindTable(tableID uint, sub string) entity.GameTable
}

type tablesService struct {
	tableRepo TablesRepository
}

func NewTablesService(repo TablesRepository) TablesService {
	return &tablesService{tableRepo: repo}
}

func (t tablesService) FindTables(gameID uint, sub string) []entity.GameTable {
	//TODO implement me
	panic("implement me")
}

func (t tablesService) FindTable(tableID uint, sub string) entity.GameTable {
	//TODO implement me
	panic("implement me")
}
