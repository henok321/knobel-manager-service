package player

type Player struct {
	ID     uint   `json:"id" gorm:"primaryKey"`
	Name   string `json:"name" gorm:"size:255 not null"`
	TeamID uint   `json:"teamID" gorm:"not null"`
}

type Team struct {
	ID      uint     `gorm:"primaryKey"`
	Name    string   `gorm:"size:255"`
	GameID  uint     `gorm:"not null"`
	members []Player `gorm:"foreignKey:TeamID;constraint:OnDelete:CASCADE;"` //nolint:unused
}
