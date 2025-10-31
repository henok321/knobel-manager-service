package health

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// DatabaseChecker checks database connectivity using GORM.
type DatabaseChecker struct {
	db      *gorm.DB
	timeout time.Duration
}

// NewDatabaseChecker creates a new database health checker with the specified timeout.
// The recommended timeout is 500ms to keep readiness probes fast.
func NewDatabaseChecker(db *gorm.DB, timeout time.Duration) *DatabaseChecker {
	return &DatabaseChecker{
		db:      db,
		timeout: timeout,
	}
}

// Name returns the identifier for this checker.
func (c *DatabaseChecker) Name() string {
	return "database"
}

// Check performs a database ping with a timeout.
// This validates that GORM and the underlying database connection are available.
// Returns nil if the database is reachable, error otherwise.
func (c *DatabaseChecker) Check(ctx context.Context) error {
	// Create a context with timeout
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
