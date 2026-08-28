package audit

import (
	"context"

	"gorm.io/gorm"

	"github.com/henok321/knobel-manager-service/pkg/entity"
)

type EventsRepository struct {
	db *gorm.DB
}

func NewEventsRepository(db *gorm.DB) *EventsRepository {
	return &EventsRepository{db}
}

func (r *EventsRepository) Insert(ctx context.Context, events []entity.AuditEvent) error {
	if len(events) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Create(&events).Error
}

// ponytail: returns a game's whole history, unbounded. Fine at tournament size; the
// (game_id, id DESC) index already supports adding limit plus a keyset cursor when a
// game outgrows a few hundred events.
func (r *EventsRepository) FindByGameID(ctx context.Context, gameID int) ([]entity.AuditEvent, error) {
	var events []entity.AuditEvent

	err := r.db.WithContext(ctx).
		Where("game_id = ?", gameID).
		Order("id DESC").
		Find(&events).Error
	if err != nil {
		return nil, err
	}

	return events, nil
}
