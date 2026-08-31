package audit

import (
	"context"

	"github.com/henok321/knobel-manager-service/pkg/apperror"
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

func (s *EventsService) FindByGameID(ctx context.Context, gameID int, sub string) ([]entity.AuditEvent, error) {
	exists, err := s.games.Exists(ctx, gameID)
	if err != nil {
		return nil, err
	}

	if exists {
		isOwner, err := s.games.IsOwner(ctx, gameID, sub)
		if err != nil {
			return nil, err
		}

		if !isOwner {
			return nil, apperror.ErrNotOwner
		}

		return s.events.FindByGameID(ctx, gameID)
	}

	wasOwner, err := s.events.WasOwner(ctx, gameID, sub)
	if err != nil {
		return nil, err
	}

	if !wasOwner {
		return nil, apperror.ErrGameNotFound
	}

	return s.events.FindByGameID(ctx, gameID)
}
