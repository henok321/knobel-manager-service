package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/henok321/knobel-manager-service/api/middleware"
	"github.com/henok321/knobel-manager-service/pkg/entity"
)

const auditWriteTimeout = 5 * time.Second

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
				snapshot, err := service.snapshot(ctx, gameID)
				if err != nil {
					slog.ErrorContext(ctx, "Could not snapshot game before mutation, skipping audit", "error", err)
					next.ServeHTTP(writer, request)

					return
				}

				before = snapshot
			}

			// POST /games is the only mutating route without gameID in its path, so its
			// new id has to come out of the response body.
			recorder := &responseRecorder{ResponseWriter: writer, status: http.StatusOK}
			if !hasGameID {
				recorder.body = &bytes.Buffer{}
			}

			// Deferred so a handler that commits and then panics still leaves a trail.
			// An audit log that silently disagrees with the database is worse than none.
			defer record(ctx, service, request, recorder, *actor, gameID, hasGameID, before)

			next.ServeHTTP(recorder, request)
		})
	}
}

func record(
	ctx context.Context,
	service *EventsService,
	request *http.Request,
	recorder *responseRecorder,
	actor middleware.User,
	gameID int,
	hasGameID bool,
	before map[recordKey]fields,
) {
	// ponytail: successful requests only. A denied attempt on someone else's game
	// leaves no trace; add an outcome column if that ever needs to be visible.
	if !recorder.succeeded() {
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

	// The mutation is already committed, so the audit write must not die with the
	// client connection: net/http cancels the request context the moment the caller
	// hangs up, which would drop the record of a change that actually happened.
	auditCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), auditWriteTimeout)
	defer cancel()

	after, err := service.snapshot(auditCtx, gameID)
	if err != nil {
		slog.ErrorContext(auditCtx, "Could not snapshot game after mutation, skipping audit", "error", err)
		return
	}

	// A game that still exists always yields at least its own record, so an empty
	// snapshot means the game was deleted and its audit rows cascaded away with it.
	// Writing now would only violate the foreign key. Checking the data rather than
	// the route keeps this correct if the paths are ever renamed or prefixed.
	if len(after) == 0 {
		return
	}

	changes := diff(before, after)

	if isSetupRoute(request.Pattern) {
		changes = append([]entityChange{{
			Entity:   entity.AuditEntityGame,
			EntityID: strconv.Itoa(gameID),
			Action:   entity.AuditActionSetup,
			Changes:  []fieldChange{},
		}}, changes...)
	}

	requestID := ""
	if requestContext, ok := middleware.RequestFromContext(ctx); ok {
		requestID = requestContext.ID
	}

	if err := service.record(auditCtx, gameID, actor, requestID, changes); err != nil {
		slog.ErrorContext(auditCtx, "Could not write audit events", "error", err)
	}
}

// Matched on the suffix, not the whole pattern: a BaseURL prefix or a path rename
// would silently stop the setup event from being recorded.
func isSetupRoute(pattern string) bool {
	return strings.HasSuffix(pattern, "/setup")
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

// Embedding hides Flusher, Hijacker and friends from anything further in; Unwrap is
// how http.ResponseController reaches the real writer again.
func (r *responseRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

func (r *responseRecorder) succeeded() bool {
	return r.status >= http.StatusOK && r.status < http.StatusMultipleChoices
}
