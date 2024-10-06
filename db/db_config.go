package db

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"knobel-manager-service/models"
	"os"
)

func Connect() (*gorm.DB, error) {
	url := os.Getenv("DATABASE_URL")
	db, err := gorm.Open(postgres.Open(url), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	db.AutoMigrate(&models.Player{}, &models.Team{}, &models.GameTable{})
	return db, nil
}
