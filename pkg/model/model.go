package model

import (
	"time"

	"gorm.io/gorm"
)

type GameStatus string

const (
	StatusSetup      GameStatus = "setup"
	StatusInProgress GameStatus = "in_progress"
	StatusCompleted  GameStatus = "completed"
)

type Game struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	Name           string         `gorm:"size:255;not null" json:"name"`
	TeamSize       uint           `gorm:"not null" json:"team_size"`
	TableSize      uint           `gorm:"not null" json:"table_size"`
	NumberOfRounds uint           `gorm:"not null" json:"number_of_rounds"`
	Status         GameStatus     `gorm:"size:50;not null" json:"status"`
	Owners         []*Owner       `gorm:"many2many:game_owners" json:"owners,omitempty"`
	Teams          []*Team        `gorm:"foreignKey:GameID" json:"teams,omitempty"`
	Rounds         []*Round       `gorm:"foreignKey:GameID" json:"rounds,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

type Owner struct {
	ID    uint    `gorm:"primaryKey" json:"id"`
	Sub   string  `gorm:"not null;unique" json:"sub"`
	Games []*Game `gorm:"many2many:game_owners;constraint:OnDelete:CASCADE,OnUpdate:CASCADE" json:"games,omitempty"`
}

type Team struct {
	ID      uint      `gorm:"primaryKey" json:"id"`
	Name    string    `gorm:"size:255;not null" json:"name"`
	GameID  uint      `gorm:"not null" json:"game_id"`
	Game    *Game     `gorm:"foreignKey:GameID" json:"game,omitempty"`
	Players []*Player `gorm:"foreignKey:TeamID" json:"players,omitempty"`
}

type Player struct {
	ID     uint     `gorm:"primaryKey" json:"id"`
	Name   string   `gorm:"size:255;not null" json:"name"`
	TeamID uint     `gorm:"not null" json:"team_id"`
	Team   *Team    `gorm:"foreignKey:TeamID" json:"team,omitempty"`
	Scores []*Score `gorm:"foreignKey:PlayerID" json:"scores,omitempty"`
}

type Round struct {
	ID          uint     `gorm:"primaryKey" json:"id"`
	RoundNumber uint     `gorm:"not null;uniqueIndex:idx_game_round" json:"round_number"`
	GameID      uint     `gorm:"not null;uniqueIndex:idx_game_round" json:"game_id"`
	Status      string   `gorm:"size:50;not null" json:"status"`
	Tables      []*Table `gorm:"foreignKey:RoundID" json:"tables,omitempty"`
}

type Table struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	TableNumber uint      `gorm:"not null;uniqueIndex:idx_round_table" json:"table_number"`
	RoundID     uint      `gorm:"not null;uniqueIndex:idx_round_table" json:"round_id"`
	Players     []*Player `gorm:"many2many:table_players" json:"players,omitempty"`
	Scores      []*Score  `gorm:"foreignKey:TableID" json:"scores,omitempty"`
}

type Score struct {
	ID       uint    `gorm:"primaryKey" json:"id"`
	PlayerID uint    `gorm:"not null;uniqueIndex:idx_player_table" json:"player_id"`
	TableID  uint    `gorm:"not null;uniqueIndex:idx_player_table" json:"table_id"`
	Score    int     `gorm:"not null" json:"score"`
	Player   *Player `gorm:"foreignKey:PlayerID" json:"player,omitempty"`
	Table    *Table  `gorm:"foreignKey:TableID" json:"table,omitempty"`
}
