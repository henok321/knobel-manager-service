package game

import "gorm.io/gorm"

func InitializeGameModule(db *gorm.DB) GamesHandler {
	repository := NewGamesRepository(db)
	service := NewGamesService(repository)
	handler := NewGamesHandler(service)
	return handler
}
