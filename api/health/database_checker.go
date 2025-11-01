package health

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type DatabaseChecker struct {
	db      *gorm.DB
	timeout time.Duration
}

func NewDatabaseChecker(db *gorm.DB, timeout time.Duration) *DatabaseChecker {
	return &DatabaseChecker{
		db:      db,
		timeout: timeout,
	}
}

func (c *DatabaseChecker) Name() string {
	return "database"
}

func (c *DatabaseChecker) Check(ctx context.Context) error {
	checkCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Get the underlying sql.DB from GORM
	sqlDB, err := c.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	// Perform the ping
	if err := sqlDB.PingContext(checkCtx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}
