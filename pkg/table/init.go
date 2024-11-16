package table

import "gorm.io/gorm"

func InitializeTablesModule(db *gorm.DB) TablesService {
	return NewTablesService(NewTablesRepository(db))
}
