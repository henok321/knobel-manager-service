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

// ponytail: returns a game's whole history, unbounded. Fine at tournament size; the
// (game_id, id DESC) index already supports adding a limit plus a keyset cursor.
func (r *EventsRepository) FindByGameID(ctx context.Context, gameID int) ([]entity.AuditEvent, error) {
	var events []entity.AuditEvent

	if err := r.db.WithContext(ctx).Where("game_id = ?", gameID).Order("id DESC").Find(&events).Error; err != nil {
		return nil, err
	}

	return events, nil
}
