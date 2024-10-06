package db

import (
	"github.com/henok321/knobel-manager-service/pkg/player"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"os"
)

func Connect() (*gorm.DB, error) {
	url := os.Getenv("DATABASE_URL")
	db, err := gorm.Open(postgres.Open(url), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	db.AutoMigrate(&player.Player{})
	return db, nil
}
