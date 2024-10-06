package player

type Player struct {
	ID     uint   `gorm:"primaryKey"`
	Name   string `gorm:"size:255"`
	TeamID uint
}
