package audit

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/henok321/knobel-manager-service/api/middleware"
)

// Route patterns needing special treatment. Everything else is handled by the
// plain before/after diff.
const (
	createGamePattern = "POST /games"
	deleteGamePattern = "DELETE /games/{gameID}"
	setupGamePattern  = "POST /games/{gameID}/setup"
)

// Middleware records who changed what for every successful mutation. It lives here
// rather than in api/middleware because pkg/game imports that package, so the
// reverse dependency would be an import cycle.
//
// Audit writes are best effort: the mutation has already been committed by the time
// this runs, so a failure is logged and never surfaces to the caller. A broken audit
// table must not break a tournament.
func Middleware(service *EventsService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx := request.Context()

			actor, hasActor := middleware.UserFromContext(ctx)

			if !mutates(request.Method) || !hasActor {
				next.ServeHTTP(writer, request)
				return
			}

			gameID, hasGameID := pathGameID(request)

			var before map[recordKey]fields
			if hasGameID {
				before = service.snapshot(ctx, gameID)
			}

			recorder := &responseRecorder{ResponseWriter: writer, status: http.StatusOK}
			if request.Pattern == createGamePattern {
				recorder.body = &bytes.Buffer{}
			}

			next.ServeHTTP(recorder, request)

			// A successful game deletion has already cascaded its audit rows away, and
			// the foreign key would reject a new one.
			if !recorder.succeeded() || request.Pattern == deleteGamePattern {
				return
			}

			if !hasGameID {
				created, err := createdGameID(recorder.body)
				if err != nil {
					slog.ErrorContext(ctx, "Could not determine created game for audit", "error", err)
					return
				}

				gameID = created
			}

			changes := diff(before, service.snapshot(ctx, gameID))

			if request.Pattern == setupGamePattern {
				changes = append([]EntityChange{setupChange(gameID)}, changes...)
			}

			requestID := ""
			if requestContext, ok := middleware.RequestFromContext(ctx); ok {
				requestID = requestContext.ID
			}

			if err := service.record(ctx, gameID, *actor, requestID, changes); err != nil {
				slog.ErrorContext(ctx, "Could not write audit events", "error", err)
			}
		})
	}
}

func mutates(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func pathGameID(request *http.Request) (int, bool) {
	gameID, err := strconv.Atoi(request.PathValue("gameID"))
	if err != nil {
		return 0, false
	}

	return gameID, true
}

// POST /games is the only mutating route without gameID in its path, so the new id
// has to come out of the response body.
func createdGameID(body *bytes.Buffer) (int, error) {
	if body == nil {
		return 0, errors.New("no response body captured")
	}

	var response struct {
		Game struct {
			ID int `json:"id"`
		} `json:"game"`
	}

	if err := json.Unmarshal(body.Bytes(), &response); err != nil {
		return 0, err
	}

	if response.Game.ID == 0 {
		return 0, errors.New("response carries no game id")
	}

	return response.Game.ID, nil
}

type responseRecorder struct {
	http.ResponseWriter
	status int
	body   *bytes.Buffer
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(body []byte) (int, error) {
	if r.body != nil {
		r.body.Write(body)
	}

	return r.ResponseWriter.Write(body)
}

func (r *responseRecorder) succeeded() bool {
	return r.status >= http.StatusOK && r.status < http.StatusMultipleChoices
}
