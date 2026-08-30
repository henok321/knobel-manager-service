package audit

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/henok321/knobel-manager-service/api/middleware"
)

type ActorPlugin struct{}

func (ActorPlugin) Name() string {
	return "audit:actor"
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

func (ActorPlugin) Initialize(db *gorm.DB) error {
	// Anchored before the write, not after "gorm:begin_transaction": at that point
	// Statement.ConnPool is still the *sql.DB pool, so the setting lands on an
	// arbitrary connection and the audit trigger records every change as "system".
	if err := db.Callback().Create().Before("gorm:create").Register("audit:actor", setActor); err != nil {
		return fmt.Errorf("register audit actor on create: %w", err)
	}

	if err := db.Callback().Update().Before("gorm:update").Register("audit:actor", setActor); err != nil {
		return fmt.Errorf("register audit actor on update: %w", err)
	}

	if err := db.Callback().Delete().Before("gorm:delete").Register("audit:actor", setActor); err != nil {
		return fmt.Errorf("register audit actor on delete: %w", err)
	}

	return nil
}
