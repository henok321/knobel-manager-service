package models

type GameTable struct {
	ID      uint     `gorm:"primaryKey"`
	Name    string   `gorm:"size:255"`
	Round   uint     `gorm:"default:0"`
	Players []Player `gorm:"many2many:game_table_players;"`
}
