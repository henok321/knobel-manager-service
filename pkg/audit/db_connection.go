package audit

import (
	"errors"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/henok321/knobel-manager-service/api/middleware"
)

func OpenDatabase(databaseURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := errors.Join(
		db.Callback().Create().Before("gorm:create").Register("audit:actor", setActor),
		db.Callback().Update().Before("gorm:update").Register("audit:actor", setActor),
		db.Callback().Delete().Before("gorm:delete").Register("audit:actor", setActor),
	); err != nil {
		return nil, fmt.Errorf("register audit actor callbacks: %w", err)
	}

	return db, nil
}

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
