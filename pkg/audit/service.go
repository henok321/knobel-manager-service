package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/henok321/knobel-manager-service/api/middleware"
	"github.com/henok321/knobel-manager-service/pkg/entity"
	"github.com/henok321/knobel-manager-service/pkg/game"
)

type EventsService struct {
	events *EventsRepository
	games  *game.GamesRepository
}

func NewEventsService(events *EventsRepository, games *game.GamesRepository) *EventsService {
	return &EventsService{events: events, games: games}
}

func (s *EventsService) FindByGameID(ctx context.Context, gameID int) ([]entity.AuditEvent, error) {
	return s.events.FindByGameID(ctx, gameID)
}

// snapshot deliberately skips the ownership check: the middleware is not an
// authorization boundary, and events are only written once the handler has
// answered 2xx.
//
// A missing game is an empty snapshot, but any other failure must be reported.
// Treating a failed query as "nothing there" would make the diff read as though
// every team, player and score had just been deleted, and that fabricated trail
// would commit happily.
func (s *EventsService) snapshot(ctx context.Context, gameID int) (map[recordKey]fields, error) {
	gameByID, err := s.games.FindByID(ctx, gameID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return map[recordKey]fields{}, nil
		}

		return nil, fmt.Errorf("cannot snapshot game %d: %w", gameID, err)
	}

	return flatten(gameByID), nil
}

func (s *EventsService) record(ctx context.Context, gameID int, actor middleware.User, requestID string, changes []entityChange) error {
	events := make([]entity.AuditEvent, 0, len(changes))

	for _, change := range changes {
		encoded, err := json.Marshal(change.Changes)
		if err != nil {
			return fmt.Errorf("cannot encode audit changes for %s %s: %w", change.Entity, change.EntityID, err)
		}

		// ponytail: actor_email is snapshotted, not resolved on read. That is the
		// historical truth and avoids a Firebase lookup per event; it goes stale if a
		// user changes their address, which is the right trade for an audit trail.
		events = append(events, entity.AuditEvent{
			GameID:     gameID,
			RequestID:  requestID,
			ActorSub:   actor.Sub,
			ActorEmail: actor.Email,
			Action:     change.Action,
			Entity:     change.Entity,
			EntityID:   change.EntityID,
			Changes:    string(encoded),
		})
	}

	return s.events.Insert(ctx, events)
}
