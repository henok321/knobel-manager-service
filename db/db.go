package db

import (
	"database/sql"
	"fmt"
	"knobel-manager-service/models"
	"log"
	"os"
)

type DB struct {
	Conn *sql.DB
}

func NewDB() (*DB, error) {
	dsn := fmt.Sprintf("%s:%s@cloudsql(%s)/%s",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("CLOUD_SQL_CONNECTION_NAME"),
		os.Getenv("DB_NAME"))

	dbConn, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create the database connection pool: %v", err)
	}

	if err := dbConn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping the database: %v", err)
	}

	log.Println("Connected to the Cloud SQL database!")
	return &DB{Conn: dbConn}, nil
}

func (db *DB) Close() {
	err := db.Conn.Close()
	if err != nil {
		return
	}
}

func (db *DB) GetExampleData() (*models.ExampleData, error) {
	var id int
	var message string
	query := "SELECT id, message FROM test LIMIT 1"
	if err := db.Conn.QueryRow(query).Scan(&id, &message); err != nil {
		return nil, fmt.Errorf("failed to query data: %v", err)
	}
	return &models.ExampleData{ID: id, Message: message}, nil
}
