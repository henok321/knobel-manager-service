package entity

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

var ErrorGameNotFound = errors.New("game not found")
var ErrorTeamNotFound = errors.New("team not found")
var ErrorNotOwner = errors.New("user is not the owner of the game")

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
	ID             uint           `gorm:"primaryKey" json:"id"`
	Name           string         `gorm:"size:255;not null" json:"name"`
	TeamSize       uint           `gorm:"not null" json:"teamSize"`
	TableSize      uint           `gorm:"not null" json:"tableSize"`
	NumberOfRounds uint           `gorm:"not null" json:"numberOfRounds"`
	Status         GameStatus     `gorm:"size:50;not null" json:"status"`
	Owners         []*GameOwner   `gorm:"foreignKey:GameID;constraint:OnDelete:CASCADE" json:"owners,omitempty"`
	Teams          []*Team        `gorm:"foreignKey:GameID" json:"team,omitempty"`
	Rounds         []*Round       `gorm:"foreignKey:GameID" json:"rounds,omitempty"`
	CreatedAt      time.Time      `json:"-"`
	UpdatedAt      time.Time      `json:"-"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

type GameOwner struct {
	GameID   uint   `gorm:"primaryKey" json:"gameID"`
	OwnerSub string `gorm:"primaryKey;size:255;not null" json:"ownerSub"`
}

type Team struct {
	ID      uint      `gorm:"primaryKey" json:"id"`
	Name    string    `gorm:"size:255;not null" json:"name"`
	GameID  uint      `gorm:"not null" json:"gameID"`
	Game    *Game     `gorm:"foreignKey:GameID" json:"game,omitempty"`
	Players []*Player `gorm:"foreignKey:TeamID" json:"players,omitempty"`
}

type Player struct {
	ID     uint     `gorm:"primaryKey" json:"id"`
	Name   string   `gorm:"size:255;not null" json:"name"`
	TeamID uint     `gorm:"not null" json:"teamID"`
	Team   *Team    `gorm:"foreignKey:TeamID" json:"team,omitempty"`
	Scores []*Score `gorm:"foreignKey:PlayerID" json:"scores,omitempty"`
}

type Round struct {
	ID          uint         `gorm:"primaryKey" json:"id"`
	RoundNumber uint         `gorm:"not null;uniqueIndex:idx_game_round" json:"roundNumber"`
	GameID      uint         `gorm:"not null;uniqueIndex:idx_game_round" json:"gameID"`
	Status      string       `gorm:"size:50;not null" json:"status"`
	Tables      []*GameTable `gorm:"foreignKey:RoundID" json:"tables,omitempty"`
}

type GameTable struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	TableNumber uint      `gorm:"not null;uniqueIndex:idx_round_table" json:"tableNumber"`
	RoundID     uint      `gorm:"not null;uniqueIndex:idx_round_table" json:"roundID"`
	Players     []*Player `gorm:"many2many:table_players" json:"players,omitempty"`
	Scores      []*Score  `gorm:"foreignKey:TableID" json:"scores,omitempty"`
}

// **Add TableName() method only for GameTable, since the struct name doesn't pluralize to 'game_tables'**
func (GameTable) TableName() string {
	return "game_tables"
}

type Score struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	PlayerID  uint       `gorm:"not null;uniqueIndex:idx_player_table" json:"playerID"`
	TableID   uint       `gorm:"not null;uniqueIndex:idx_player_table" json:"tableID"`
	Score     int        `gorm:"not null" json:"score"`
	Player    *Player    `gorm:"foreignKey:PlayerID" json:"player,omitempty"`
	GameTable *GameTable `gorm:"foreignKey:TableID" json:"gameTable,omitempty"`
}

type TablePlayer struct {
	TableID  uint `gorm:"primaryKey" json:"tableID"`
	PlayerID uint `gorm:"primaryKey" json:"playerID"`
}

// **Add TableName() method for TablePlayer, since the default pluralization might not match 'table_players'**
func (TablePlayer) TableName() string {
	return "table_players"
}
