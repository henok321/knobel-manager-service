package player

import (
	"gorm.io/gorm"
)

func InitializePlayerModule(db *gorm.DB) PlayersService {
	repository := NewPlayerRepository(db)
	service := NewPlayersService(repository)
	return service
}
