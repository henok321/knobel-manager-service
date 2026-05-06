package team

import (
	"gorm.io/gorm"

	"github.com/henok321/knobel-manager-service/pkg/game"
)

func InitializeTeamsModule(db *gorm.DB) TeamsService {
	return NewTeamsService(NewTeamsRepository(db), game.NewGamesRepository(db))
}
