package entity

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

var ErrorGameNotFound = errors.New("game not found")
var ErrorTeamNotFound = errors.New("team not found")
var ErrorPlayerNotFound = errors.New("player not found")
var ErrorNotOwner = errors.New("user is not the owner of the requested resource")
var ErrorTableAssignment = errors.New("cannot not assign players to tables")
var ErrorInvalidScore = errors.New("invalid score")
var ErrorRoundOrTableNotFound = errors.New("round or table not found")

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
	Teams          []*Team        `gorm:"foreignKey:GameID" json:"team,omitempty"`
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
	ID      int       `gorm:"primaryKey" json:"id"`
	Name    string    `gorm:"size:255;not null" json:"name"`
	GameID  int       `gorm:"not null" json:"gameID"`
	Game    *Game     `gorm:"foreignKey:GameID" json:"game,omitempty"`
	Players []*Player `gorm:"foreignKey:TeamID" json:"players,omitempty"`
}

type Player struct {
	ID     int      `gorm:"primaryKey" json:"id"`
	Name   string   `gorm:"size:255;not null" json:"name"`
	TeamID int      `gorm:"not null" json:"teamID"`
	Team   *Team    `gorm:"foreignKey:TeamID" json:"team,omitempty"`
	Scores []*Score `gorm:"foreignKey:PlayerID" json:"scores,omitempty"`
}

type Round struct {
	ID          int          `gorm:"primaryKey" json:"id"`
	RoundNumber int          `gorm:"not null;uniqueIndex:idx_game_round" json:"roundNumber"`
	GameID      int          `gorm:"not null;uniqueIndex:idx_game_round" json:"gameID"`
	Status      string       `gorm:"size:50;not null" json:"status"`
	Tables      []*GameTable `gorm:"foreignKey:RoundID" json:"table,omitempty"`
}

type GameTable struct {
	ID          int       `gorm:"primaryKey" json:"id"`
	TableNumber int       `gorm:"not null;uniqueIndex:idx_round_table" json:"tableNumber"`
	RoundID     int       `gorm:"not null;uniqueIndex:idx_round_table" json:"roundID"`
	Players     []*Player `gorm:"many2many:table_players" json:"players,omitempty"`
	Scores      []*Score  `gorm:"foreignKey:TableID" json:"scores,omitempty"`
}

// **Add TableName() method only for GameTable, since the struct name doesn't pluralize to 'game_tables'**
func (GameTable) TableName() string {
	return "game_tables"
}

type Score struct {
	ID        int        `gorm:"primaryKey" json:"id"`
	PlayerID  int        `gorm:"not null;uniqueIndex:idx_player_table" json:"playerID"`
	TableID   int        `gorm:"not null;uniqueIndex:idx_player_table" json:"tableID"`
	Score     int        `gorm:"not null" json:"score"`
	Players   []*Player  `gorm:"many2many:table_players" json:"players,omitempty"`
	GameTable *GameTable `gorm:"foreignKey:TableID" json:"gameTable,omitempty"`
}

type TablePlayer struct {
	TableID  int `gorm:"primaryKey;column:game_table_id" json:"tableID"`
	PlayerID int `gorm:"primaryKey;column:player_id" json:"playerID"`
}

// **Add TableName() method for TablePlayer, since the default pluralization might not setup 'table_players'**
func (TablePlayer) TableName() string {
	return "table_players"
}
