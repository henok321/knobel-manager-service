package models

type Team struct {
	ID      uint   `gorm:"primaryKey"`
	Name    string `gorm:"size:255"`
	Players []Player
}
