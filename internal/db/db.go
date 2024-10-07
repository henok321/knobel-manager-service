package db

import (
	"log"
	"os"

	"github.com/henok321/knobel-manager-service/pkg/game"
	"github.com/henok321/knobel-manager-service/pkg/player"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect() (*gorm.DB, error) {
	url := os.Getenv("DATABASE_URL")
	db, err := gorm.Open(postgres.Open(url), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&player.Player{}, &game.Game{}, &game.Owner{})

	if err != nil {
		log.Fatalf("error migrating database: %v", err)
	}
	return db, nil
}
