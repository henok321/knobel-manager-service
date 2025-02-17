package game

import "gorm.io/gorm"

func InitializeGameModule(db *gorm.DB) GamesService {
	return NewGamesService(NewGamesRepository(db))
}
