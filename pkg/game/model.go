package game

import "github.com/henok321/knobel-manager-service/pkg/player"

type Game struct {
	ID     uint          `json:"id" gorm:"primaryKey"`
	Name   string        `json:"name" gorm:"not null"`
	Teams  []player.Team `json:"teams,omitempty" gorm:"foreignKey:GameID;constraint:OnDelete:CASCADE;gorm:default:[]"`
	Owners []Owner       `json:"owners,omitempty" gorm:"many2many:game_owners;constraint:OnDelete:CASCADE,OnUpdate:CASCADE;gorm:default:[]"`
}

type Owner struct {
	ID  uint   `json:"id" gorm:"primaryKey"`
	Sub string `json:"sub" gorm:"not null,unique"`
}
