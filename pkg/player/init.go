package player

import (
	"github.com/henok321/knobel-manager-service/pkg/team"
	"gorm.io/gorm"
)

func InitializePlayerModule(db *gorm.DB) *PlayersService {
	return NewPlayersService(NewPlayersRepository(db), team.NewTeamsRepository(db))
}
