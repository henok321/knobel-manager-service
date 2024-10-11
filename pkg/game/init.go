package game

import "gorm.io/gorm"

func InitializeGameModule(db *gorm.DB) GamesService {
	repository := NewGamesRepository(db)
	service := NewGamesService(repository)

	return service
}
