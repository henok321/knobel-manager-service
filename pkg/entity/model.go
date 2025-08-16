package entity

import (
	"errors"
	"time"

	"gorm.io/gorm"
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
	ID             int            `gorm:"primaryKey" json:"id"`
	Name           string         `gorm:"size:255;not null" json:"name"`
	TeamSize       int            `gorm:"not null" json:"teamSize"`
	TableSize      int            `gorm:"not null" json:"tableSize"`
	NumberOfRounds int            `gorm:"not null" json:"numberOfRounds"`
	Status         GameStatus     `gorm:"size:50;not null" json:"status"`
	Owners         []*GameOwner   `gorm:"foreignKey:GameID;constraint:OnDelete:CASCADE" json:"owners,omitempty"`
	Teams          []*Team        `gorm:"foreignKey:GameID" json:"teams,omitempty"`
	Rounds         []*Round       `gorm:"foreignKey:GameID" json:"rounds,omitempty"`
	CreatedAt      time.Time      `json:"-"`
	UpdatedAt      time.Time      `json:"-"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

type GameOwner struct {
	GameID   int    `gorm:"primaryKey" json:"gameID"`
	OwnerSub string `gorm:"primaryKey;size:255;not null" json:"ownerSub"`
}

type ActiveGame struct {
	OwnerSub string `gorm:"primaryKey;size:255;not null" json:"ownerSub"`
	GameID   int    `gorm:"not null" json:"gameID"`
}

type Team struct {
	ID        int            `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"size:255;not null" json:"name"`
	GameID    int            `gorm:"not null" json:"gameID"`
	Game      *Game          `gorm:"foreignKey:GameID" json:"-"`
	Players   []*Player      `gorm:"foreignKey:TeamID" json:"players,omitempty"`
	CreatedAt time.Time      `json:"-"`
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type Player struct {
	ID        int            `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"size:255;not null" json:"name"`
	TeamID    int            `gorm:"not null" json:"teamID"`
	Team      *Team          `gorm:"foreignKey:TeamID" json:"-"`
	Scores    []*Score       `gorm:"foreignKey:PlayerID" json:"-"`
	CreatedAt time.Time      `json:"-"`
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type Round struct {
	ID          int            `gorm:"primaryKey" json:"id"`
	RoundNumber int            `gorm:"not null;uniqueIndex:idx_game_round" json:"roundNumber"`
	GameID      int            `gorm:"not null;uniqueIndex:idx_game_round" json:"gameID"`
	Status      string         `gorm:"size:50;not null" json:"status"`
	Tables      []*GameTable   `gorm:"foreignKey:RoundID" json:"tables,omitempty"`
	CreatedAt   time.Time      `json:"-"`
	UpdatedAt   time.Time      `json:"-"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

type GameTable struct {
	ID          int            `gorm:"primaryKey" json:"id"`
	TableNumber int            `gorm:"not null;uniqueIndex:idx_round_table" json:"tableNumber"`
	RoundID     int            `gorm:"not null;uniqueIndex:idx_round_table" json:"roundID"`
	Players     []*Player      `gorm:"many2many:table_players" json:"players,omitempty"`
	Scores      []*Score       `gorm:"foreignKey:TableID" json:"scores,omitempty"`
	CreatedAt   time.Time      `json:"-"`
	UpdatedAt   time.Time      `json:"-"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (GameTable) TableName() string {
	return "game_tables"
}

type Score struct {
	ID        int            `gorm:"primaryKey" json:"id"`
	PlayerID  int            `gorm:"not null;uniqueIndex:idx_player_table" json:"playerID"`
	TableID   int            `gorm:"not null;uniqueIndex:idx_player_table" json:"tableID"`
	Score     int            `gorm:"not null" json:"score"`
	Players   []*Player      `gorm:"many2many:table_players" json:"-"`
	GameTable *GameTable     `gorm:"foreignKey:TableID" json:"-"`
	CreatedAt time.Time      `json:"-"`
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type TablePlayer struct {
	TableID  int `gorm:"primaryKey;column:game_table_id" json:"tableID"`
	PlayerID int `gorm:"primaryKey;column:player_id" json:"playerID"`
}

func (TablePlayer) TableName() string {
	return "table_players"
}
