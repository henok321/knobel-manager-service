package db

import (
	"log"

	"github.com/henok321/knobel-manager-service/pkg/game"
	"github.com/henok321/knobel-manager-service/pkg/player"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func ConnectTest() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&player.Player{}, &game.Game{}, &game.Owner{})
	if err != nil {
		log.Fatalf("error migrating database: %v", err)
	}
	return db, nil
}
