package player

import (
	"gorm.io/gorm"

	"github.com/henok321/knobel-manager-service/pkg/team"
)

func InitializePlayerModule(db *gorm.DB) PlayersService {
	return NewPlayersService(NewPlayersRepository(db), team.NewTeamsRepository(db))
}
