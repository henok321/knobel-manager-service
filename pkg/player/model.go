package player

type Player struct {
	ID     uint   `gorm:"primaryKey"`
	Name   string `gorm:"size:255 not null"`
	TeamID uint
	Team   Team `gorm:"constraint:OnDelete:CASCADE;"`
}

type Team struct {
	ID      uint     `gorm:"primaryKey"`
	Name    string   `gorm:"size:255"`
	members []Player `gorm:"foreignKey:TeamID;constraint:OnDelete:CASCADE;"`
}
