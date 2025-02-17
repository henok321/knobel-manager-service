package team

import (
	"github.com/henok321/knobel-manager-service/pkg/game"
	"gorm.io/gorm"
)

func InitializeTeamsModule(db *gorm.DB) *TeamsService {
	return NewTeamsService(NewTeamsRepository(db), game.NewGamesRepository(db))
}
