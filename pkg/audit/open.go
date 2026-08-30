package audit

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// OpenDatabase is the only supported way to open a connection for this service. Opening
// with gorm.Open directly and forgetting ActorPlugin is silent: every audit event records
// "system", nothing errors, and nothing fails until someone reads the log.
func OpenDatabase(databaseURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.Use(ActorPlugin{}); err != nil {
		return nil, fmt.Errorf("register audit actor plugin: %w", err)
	}

	return db, nil
}
