package audit

import (
	"errors"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/henok321/knobel-manager-service/api/middleware"
)

// OpenDatabase is the only supported way to open a connection for this service. Opening
// with gorm.Open directly and forgetting the actor callbacks is silent: every audit event
// records "system", nothing errors, and nothing fails until someone reads the log.
func OpenDatabase(databaseURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Anchored before the write, not after "gorm:begin_transaction": at that point
	// Statement.ConnPool is still the *sql.DB pool, so the setting lands on an
	// arbitrary connection and the audit trigger records every change as "system".
	if err := errors.Join(
		db.Callback().Create().Before("gorm:create").Register("audit:actor", setActor),
		db.Callback().Update().Before("gorm:update").Register("audit:actor", setActor),
		db.Callback().Delete().Before("gorm:delete").Register("audit:actor", setActor),
	); err != nil {
		return nil, fmt.Errorf("register audit actor callbacks: %w", err)
	}

	return db, nil
}

// The settings are transaction-local, so this depends on GORM opening a transaction for
// every write — true unless SkipDefaultTransaction is set, which would make the setting
// land on a pooled connection, vanish immediately, and attribute everything to "system"
// without erroring.
func setActor(tx *gorm.DB) {
	user, ok := middleware.UserFromContext(tx.Statement.Context)
	if !ok {
		return
	}

	_, err := tx.Statement.ConnPool.ExecContext(
		tx.Statement.Context,
		"SELECT set_config('app.actor_sub', $1, true), set_config('app.actor_email', $2, true)",
		user.Sub, user.Email,
	)
	if err != nil {
		_ = tx.AddError(err)
	}
}
