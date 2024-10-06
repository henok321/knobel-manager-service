package player

import (
	"gorm.io/gorm"
)

func InitializePlayerModule(db *gorm.DB) *PlayersHandler {
	repository := NewPlayerRepository(db)
	service := NewPlayersService(repository)
	handler := NewPlayersHandler(service)
	return handler
}
