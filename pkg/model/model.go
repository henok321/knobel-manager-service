package model

type Game struct {
	ID     uint    `json:"id" gorm:"primaryKey"`
	Name   string  `json:"name" gorm:"not null"`
	Teams  []Team  `json:"teams,omitempty" gorm:"foreignKey:GameID;constraint:OnDelete:CASCADE;gorm:default:[]"`
	Owners []Owner `json:"owners,omitempty" gorm:"many2many:game_owners;constraint:OnDelete:CASCADE,OnUpdate:CASCADE;gorm:default:[]"`
}

type Owner struct {
	ID    uint   `json:"id" gorm:"primaryKey"`
	Sub   string `json:"sub" gorm:"not null,unique"`
	Games []Game `json:"games,omitempty" gorm:"many2many:game_owners;constraint:OnDelete:CASCADE,OnUpdate:CASCADE;gorm:default:[]"`
}

type Player struct {
	ID     uint   `json:"id" gorm:"primaryKey"`
	Name   string `json:"name" gorm:"size:255 not null"`
	TeamID uint   `json:"teamID" gorm:"not null"`
	Team   Team   `json:"-" gorm:"foreignKey:TeamID;constraint:OnDelete:CASCADE;gorm:default:[]"`
}

type Team struct {
	ID      uint     `gorm:"primaryKey"`
	Name    string   `gorm:"size:255"`
	GameID  uint     `gorm:"not null"`
	members []Player `gorm:"foreignKey:TeamID;constraint:OnDelete:CASCADE;"` //nolint:unused
}

type Round struct {
	ID          uint     `gorm:"primaryKey"`
	RoundNumber int      `gorm:"not null"`
	GameID      uint     `gorm:"not null"`
	Game        Game     `gorm:"foreignKey:GameID"`
	Tables      []*Table `gorm:"foreignKey:RoundID"`
}

type Table struct {
	ID      uint      `gorm:"primaryKey"`
	RoundID uint      `gorm:"not null"`
	Round   Round     `gorm:"foreignKey:RoundID"`
	Players []*Player `gorm:"many2many:table_players"`
}

type PlayerScore struct {
	PlayerID uint   `gorm:"primaryKey"`
	RoundID  uint   `gorm:"primaryKey"`
	Score    int    `gorm:"not null"`
	Player   Player `gorm:"foreignKey:PlayerID"`
	Round    Round  `gorm:"foreignKey:RoundID"`
}
