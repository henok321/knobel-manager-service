package player

import (
	"github.com/henok321/knobel-manager-service/pkg/team"
	"gorm.io/gorm"
)

func InitializePlayerModule(db *gorm.DB) PlayersService {
	return playersService{playersRepo: NewPlayersRepository(db), teamsRepo: team.NewTeamsRepository(db)}
}
