package entity

import (
	"errors"
	"time"
)

var ErrGameNotFound = errors.New("game not found")

type GameStatus string

const (
	StatusSetup      GameStatus = "setup"
	StatusInProgress GameStatus = "in_progress"
	StatusCompleted  GameStatus = "completed"
)

func IsOwner(game Game, sub string) bool {
	for _, owner := range game.Owners {
		if owner.OwnerSub == sub {
			return true
		}
	}

	return false
}

type Game struct {
	ID             int          `gorm:"primaryKey"`
	Name           string       `gorm:"column:game_name;size:255;not null"`
	TeamSize       int          `gorm:"not null"`
	TableSize      int          `gorm:"not null"`
	NumberOfRounds int          `gorm:"not null"`
	Status         GameStatus   `gorm:"size:50;not null"`
	Owners         []*GameOwner `gorm:"foreignKey:GameID;constraint:OnDelete:CASCADE"`
	Teams          []*Team      `gorm:"foreignKey:GameID"`
	Rounds         []*Round     `gorm:"foreignKey:GameID"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type GameOwner struct {
	GameID   int    `gorm:"primaryKey"`
	OwnerSub string `gorm:"primaryKey;size:255;not null"`
}

type Team struct {
	ID        int       `gorm:"primaryKey"`
	Name      string    `gorm:"column:team_name;size:255;not null"`
	GameID    int       `gorm:"not null"`
	Game      *Game     `gorm:"foreignKey:GameID"`
	Players   []*Player `gorm:"foreignKey:TeamID"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Player struct {
	ID        int      `gorm:"primaryKey"`
	Name      string   `gorm:"column:player_name;size:255;not null"`
	TeamID    int      `gorm:"not null"`
	Team      *Team    `gorm:"foreignKey:TeamID"`
	Scores    []*Score `gorm:"foreignKey:PlayerID"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Round struct {
	ID          int          `gorm:"primaryKey"`
	RoundNumber int          `gorm:"not null;uniqueIndex:idx_game_round"`
	GameID      int          `gorm:"not null;uniqueIndex:idx_game_round"`
	Status      string       `gorm:"size:50;not null"`
	Tables      []*GameTable `gorm:"foreignKey:RoundID"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type GameTable struct {
	ID          int       `gorm:"primaryKey"`
	TableNumber int       `gorm:"not null;uniqueIndex:idx_round_table"`
	RoundID     int       `gorm:"not null;uniqueIndex:idx_round_table"`
	Players     []*Player `gorm:"many2many:table_players"`
	Scores      []*Score  `gorm:"foreignKey:TableID"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (GameTable) TableName() string {
	return "game_tables"
}

type Score struct {
	ID        int        `gorm:"primaryKey"`
	PlayerID  int        `gorm:"not null;uniqueIndex:idx_player_table"`
	TableID   int        `gorm:"not null;uniqueIndex:idx_player_table"`
	Score     int        `gorm:"not null"`
	Players   []*Player  `gorm:"many2many:table_players"`
	GameTable *GameTable `gorm:"foreignKey:TableID"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type TablePlayer struct {
	TableID  int `gorm:"primaryKey;column:game_table_id"`
	PlayerID int `gorm:"primaryKey;column:player_id"`
}

func (TablePlayer) TableName() string {
	return "table_players"
}
