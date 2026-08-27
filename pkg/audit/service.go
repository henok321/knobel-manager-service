package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

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
// answered 2xx. A game that does not exist yields an empty snapshot.
func (s *EventsService) snapshot(ctx context.Context, gameID int) map[recordKey]fields {
	gameByID, err := s.games.FindByID(ctx, gameID)
	if err != nil {
		return map[recordKey]fields{}
	}

	return flatten(gameByID)
}

func (s *EventsService) record(ctx context.Context, gameID int, actor middleware.User, requestID string, changes []EntityChange) error {
	if len(changes) == 0 {
		return nil
	}

	events := make([]entity.AuditEvent, 0, len(changes))

	for _, change := range changes {
		encoded, err := json.Marshal(change.Changes)
		if err != nil {
			return fmt.Errorf("cannot encode audit changes for %s %s: %w", change.Entity, change.EntityID, err)
		}

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

func setupChange(gameID int) EntityChange {
	return EntityChange{
		Entity:   entity.AuditEntityGame,
		EntityID: strconv.Itoa(gameID),
		Action:   entity.AuditActionSetup,
		Changes:  []FieldChange{},
	}
}
