package audit

import (
	"context"

	"github.com/henok321/knobel-manager-service/pkg/entity"
	"github.com/henok321/knobel-manager-service/pkg/game"
)

type EventsService struct {
	events *EventsRepository
	games  *game.GamesService
}

func NewEventsService(events *EventsRepository, games *game.GamesService) *EventsService {
	return &EventsService{events: events, games: games}
}

func (s *EventsService) FindByGameID(ctx context.Context, gameID int, sub string) ([]entity.AuditEvent, error) {
	if _, err := s.games.FindByID(ctx, gameID, sub); err != nil {
		return nil, err
	}

	return s.events.FindByGameID(ctx, gameID)
}
