package db

import (
	"database/sql"
	"fmt"
	"knobel-manager-service/models"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

type DB struct {
	Conn *sql.DB
}

func NewDB() (*DB, error) {
	mustGetenv := func(k string) string {
		v := os.Getenv(k)
		if v == "" {
			log.Fatalf("Fatal Error: %s environment variable not set.", k)
		}
		return v
	}

	dbUser := mustGetenv("DB_USER")
	dbPwd := mustGetenv("DB_PASSWORD")
	dbName := mustGetenv("DB_NAME")
	unixSocketPath := mustGetenv("INSTANCE_UNIX_SOCKET")

	dbURI := fmt.Sprintf("%s:%s@unix(%s)/%s?parseTime=true",
		dbUser, dbPwd, unixSocketPath, dbName)

	dbConn, err := sql.Open("mysql", dbURI)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %v", err)
	}

	if err := dbConn.Ping(); err != nil {
		return nil, fmt.Errorf("dbConn.Ping: %v", err)
	}

	log.Println("Connected to the Cloud SQL database!")
	return &DB{Conn: dbConn}, nil
}

func (db *DB) Close() {
	if err := db.Conn.Close(); err != nil {
		log.Printf("Error closing database connection: %v", err)
	}
}

func (db *DB) GetExampleData() (*models.ExampleData, error) {
	if db.Conn == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}
	var id int
	var message string
	query := "SELECT id, message FROM test LIMIT 1"
	if err := db.Conn.QueryRow(query).Scan(&id, &message); err != nil {
		return nil, fmt.Errorf("failed to query data: %v", err)
	}
	return &models.ExampleData{ID: id, Message: message}, nil
}
