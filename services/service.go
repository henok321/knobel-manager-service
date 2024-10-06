package services

import (
	"knobel-manager-service/models"
	"log"
)

type ExampleService struct {
}

func NewExampleService() *ExampleService {
	return &ExampleService{}
}

func (s *ExampleService) SampleData() (*models.ExampleData, error) {
	log.Println("Fetching greeting from the database...")
	return &models.ExampleData{ID: 1, Message: "Hello, World!"}, nil
}
