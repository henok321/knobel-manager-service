package services

import (
	"knobel-manager-service/db"
	"knobel-manager-service/models"
	"log"
)

type ExampleService struct {
	DB *db.DB
}

func NewExampleService(db *db.DB) *ExampleService {
	return &ExampleService{DB: db}
}

func (s *ExampleService) SampleData() (*models.ExampleData, error) {
	log.Println("Fetching greeting from the database...")
	data, err := s.DB.GetExampleData()
	if err != nil {
		return nil, err
	}
	return data, nil
}
