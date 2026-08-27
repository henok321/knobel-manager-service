package integrationtests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type auditRow struct {
	action    string
	entity    string
	entityID  string
	actorSub  string
	actorMail string
	changes   string
	requestID string
}

func queryAuditEvents(t *testing.T, db *sql.DB) []auditRow {
	t.Helper()

	rows, err := db.QueryContext(t.Context(),
		"SELECT action, entity, entity_id, actor_sub, actor_email, changes, request_id FROM audit_events ORDER BY id")
	require.NoError(t, err, "cannot read audit events")

	defer rows.Close()

	events := make([]auditRow, 0)

	for rows.Next() {
		var row auditRow
		require.NoError(t, rows.Scan(&row.action, &row.entity, &row.entityID, &row.actorSub, &row.actorMail, &row.changes, &row.requestID))
		events = append(events, row)
	}

	require.NoError(t, rows.Err())

	return events
}

func findAuditEvent(t *testing.T, events []auditRow, entity, entityID string) auditRow {
	t.Helper()

	for _, event := range events {
		if event.entity == entity && event.entityID == entityID {
			return event
		}
	}

	t.Fatalf("no audit event for %s %s in %+v", entity, entityID, events)

	return auditRow{}
}

func TestAuditEventsWrittenByMutations(t *testing.T) {
	tests := map[string]testCase{
		"Create game records game and owner": {
			method:             "POST",
			endpoint:           "/games",
			requestBody:        `{"name":"Game 1","teamSize":4,"tableSize":4,"numberOfRounds":2}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusCreated,
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()

				events := queryAuditEvents(t, db)
				assert.Len(t, events, 2, "expected a game and an owner event")

				gameEvent := findAuditEvent(t, events, "game", "1")
				assert.Equal(t, "create", gameEvent.action)
				assert.Equal(t, "sub-1", gameEvent.actorSub)
				assert.JSONEq(t, `[
					{"field":"name","from":null,"to":"Game 1"},
					{"field":"number_of_rounds","from":null,"to":"2"},
					{"field":"status","from":null,"to":"setup"},
					{"field":"table_size","from":null,"to":"4"},
					{"field":"team_size","from":null,"to":"4"}
				]`, gameEvent.changes)

				ownerEvent := findAuditEvent(t, events, "owner", "sub-1")
				assert.Equal(t, "create", ownerEvent.action)
				assert.JSONEq(t, `[]`, ownerEvent.changes, "owner events are presence only")
			},
		},
		"Update team records the renamed field": {
			method:             "PUT",
			endpoint:           "/games/1/teams/1",
			requestBody:        `{"name":"Team 1 updated"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusOK,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_team.sql")
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()

				events := queryAuditEvents(t, db)
				assert.Len(t, events, 1)

				teamEvent := findAuditEvent(t, events, "team", "1")
				assert.Equal(t, "update", teamEvent.action)
				assert.JSONEq(t, `[{"field":"name","from":"Team 1","to":"Team 1 updated"}]`, teamEvent.changes)
			},
		},
		"Create team records the new team": {
			method:             "POST",
			endpoint:           "/games/1/teams",
			requestBody:        `{"name":"Team 1"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusCreated,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()

				events := queryAuditEvents(t, db)
				assert.Len(t, events, 1)

				teamEvent := findAuditEvent(t, events, "team", "1")
				assert.Equal(t, "create", teamEvent.action)
				assert.JSONEq(t, `[{"field":"name","from":null,"to":"Team 1"}]`, teamEvent.changes)
			},
		},
		"Delete team records the cascaded player too": {
			method:             "DELETE",
			endpoint:           "/games/1/teams/1",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusNoContent,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_team_player.sql")
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()

				events := queryAuditEvents(t, db)
				assert.Len(t, events, 2, "the player cascades away with the team and must be recorded")

				assert.Equal(t, "delete", findAuditEvent(t, events, "team", "1").action)

				playerEvent := findAuditEvent(t, events, "player", "1")
				assert.Equal(t, "delete", playerEvent.action)
				assert.JSONEq(t, `[
					{"field":"name","from":"Player 1","to":null},
					{"field":"team_id","from":"1","to":null}
				]`, playerEvent.changes)
			},
		},
		"Add owner records the new owner": {
			method:             "POST",
			endpoint:           "/games/1/owners",
			requestBody:        `{"email":"sub-2@example.org"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusOK,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()

				events := queryAuditEvents(t, db)
				assert.Len(t, events, 1)
				assert.Equal(t, "create", findAuditEvent(t, events, "owner", "sub-2").action)
			},
		},
		// The spec promises setup collapses to one event: rounds, tables and table
		// players are excluded from the diff, so ~130 rows never materialize.
		"Setup records a single setup event": {
			method:             "POST",
			endpoint:           "/games/1/setup",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusNoContent,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_ready.sql")
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()

				events := queryAuditEvents(t, db)
				assert.Len(t, events, 1, "rounds and tables must stay out of the log")

				setupEvent := findAuditEvent(t, events, "game", "1")
				assert.Equal(t, "setup", setupEvent.action)
				assert.JSONEq(t, `[]`, setupEvent.changes)

				var tables int
				require.NoError(t, db.QueryRowContext(t.Context(),
					"SELECT count(*) FROM game_tables").Scan(&tables))
				assert.Positive(t, tables, "setup must really have created tables")
			},
		},
		"Re-running setup records every wiped score": {
			method:             "POST",
			endpoint:           "/games/1/setup",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusNoContent,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_assigned_with_scores.sql")

				// The fixture is mid-tournament; setup only runs from the setup state.
				if _, err := db.ExecContext(t.Context(), "UPDATE games SET status = 'setup' WHERE id = 1"); err != nil {
					t.Fatalf("Failed to reset game status: %v", err)
				}
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()

				events := queryAuditEvents(t, db)

				deletedScores := 0

				for _, event := range events {
					if event.entity == "score" {
						assert.Equal(t, "delete", event.action)
						deletedScores++
					}
				}

				assert.Positive(t, deletedScores, "a setup re-run destroys scores and must say so")
				assert.Equal(t, "setup", findAuditEvent(t, events, "game", "1").action)
			},
		},
		"Update scores records the score change": {
			method:             "PUT",
			endpoint:           "/games/1/rounds/1/tables/1/scores",
			requestBody:        `{"scores": [{"playerID":1,"score":6},{"playerID":5,"score":3},{"playerID":9,"score":2},{"playerID":13,"score":1}]}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusOK,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_assigned.sql")
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()

				events := queryAuditEvents(t, db)
				assert.Len(t, events, 4, "one event per scored player")

				for _, event := range events {
					assert.Equal(t, "score", event.entity)
					assert.Equal(t, "create", event.action, "these scores did not exist before")
					assert.Equal(t, "sub-1", event.actorSub)
				}
			},
		},
		// The path gameID used to be decorative for player routes, which let any
		// caller mutate a player under a bogus game and leave no audit trail at all.
		"Player mutation under a bogus gameID is rejected": {
			method:             "PUT",
			endpoint:           "/games/999/teams/1/players/1",
			requestBody:        `{"name":"Renamed via bogus path"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusNotFound,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_team_player.sql")
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()

				var name string
				require.NoError(t, db.QueryRowContext(t.Context(),
					"SELECT player_name FROM players WHERE id = 1").Scan(&name))
				assert.Equal(t, "Player 1", name, "the player must not have been touched")
				assert.Empty(t, queryAuditEvents(t, db))
			},
		},
		"Player mutation records the change and the actor": {
			method:             "PUT",
			endpoint:           "/games/1/teams/1/players/1",
			requestBody:        `{"name":"Anna"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusOK,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_team_player.sql")
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()

				events := queryAuditEvents(t, db)
				assert.Len(t, events, 1)

				playerEvent := findAuditEvent(t, events, "player", "1")
				assert.Equal(t, "update", playerEvent.action)
				assert.JSONEq(t, `[{"field":"name","from":"Player 1","to":"Anna"}]`, playerEvent.changes)

				// The actor identity and request grouping are the whole point, so assert
				// them rather than letting an empty fallback pass silently.
				assert.Equal(t, "sub-1", playerEvent.actorSub)
				assert.Equal(t, "sub-1@example.org", playerEvent.actorMail)
				assert.NotEmpty(t, playerEvent.requestID, "events must carry the request id")
			},
		},
		"Rejected mutation records nothing": {
			method:             "POST",
			endpoint:           "/games/1/teams",
			requestBody:        `{"name":"Team 1"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-2"},
			expectedStatusCode: http.StatusForbidden,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()

				assert.Empty(t, queryAuditEvents(t, db), "only successful mutations are audited")
			},
		},
		"Invalid request body records nothing": {
			method:             "PUT",
			endpoint:           "/games/1/teams/1",
			requestBody:        `{"nope":true}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusBadRequest,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_team.sql")
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()

				assert.Empty(t, queryAuditEvents(t, db))
			},
		},
		// Audit rows cascade with the game, so the deletion cannot record itself.
		"Delete game records nothing": {
			method:             "DELETE",
			endpoint:           "/games/1",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusNoContent,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()

				assert.Empty(t, queryAuditEvents(t, db))
			},
		},
		// A missing game is a legitimate empty snapshot, distinct from a failed query.
		// Nothing is written and nothing is fabricated.
		"Mutation on a missing game records nothing": {
			method:             "PUT",
			endpoint:           "/games/999/teams/1",
			requestBody:        `{"name":"Team 1"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusNotFound,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_team.sql")
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()

				assert.Empty(t, queryAuditEvents(t, db))
			},
		},
		"Read requests record nothing": {
			method:             "GET",
			endpoint:           "/games/1",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusOK,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()

				assert.Empty(t, queryAuditEvents(t, db))
			},
		},
	}

	dbConn, teardownDatabase := setupTestDatabase(t)
	defer teardownDatabase()

	db, err := sql.Open("pgx", dbConn)
	if err != nil {
		t.Fatalf("Failed to open database connection: %v", err)
	}

	defer db.Close()

	runGooseUp(t, db)

	server, teardown := setupTestServer(t)
	defer teardown(server)

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup(db)
			}

			defer executeSQLFile(t, db, "./test_data/cleanup.sql")
			newTestRequest(t, tc, server, db)
		})
	}
}

func TestAuditLogEndpoint(t *testing.T) {
	tests := map[string]testCase{
		"Get audit log newest first": {
			method:             "GET",
			endpoint:           "/games/1/audit",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusOK,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/audit_events.sql")
			},
			expectedBody: `{"events":[
				{
					"id":2,
					"timestamp":"2026-08-27T11:00:00Z",
					"requestID":"req-bbb",
					"actor":{"sub":"sub-1","email":"owner@example.com"},
					"action":"update",
					"entity":"team",
					"entityID":"5",
					"changes":[{"field":"name","from":"Team A","to":"Team B"}]
				},
				{
					"id":1,
					"timestamp":"2026-08-27T10:00:00Z",
					"requestID":"req-aaa",
					"actor":{"sub":"sub-1","email":"owner@example.com"},
					"action":"create",
					"entity":"game",
					"entityID":"1",
					"changes":[{"field":"name","from":null,"to":"Game 1"}]
				}
			]}`,
		},
		"Get audit log empty": {
			method:             "GET",
			endpoint:           "/games/1/audit",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusOK,
			expectedBody:       `{"events":[]}`,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			},
		},
		"Get audit log not owner": {
			method:             "GET",
			endpoint:           "/games/1/audit",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-2"},
			expectedStatusCode: http.StatusForbidden,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/audit_events.sql")
			},
		},
		"Get audit log game not found": {
			method:             "GET",
			endpoint:           "/games/99/audit",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusNotFound,
		},
		"Get audit log unauthenticated": {
			method:             "GET",
			endpoint:           "/games/1/audit",
			expectedStatusCode: http.StatusUnauthorized,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/audit_events.sql")
			},
		},
		"Get audit log invalid gameID": {
			method:             "GET",
			endpoint:           "/games/abc/audit",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusBadRequest,
		},
	}

	dbConn, teardownDatabase := setupTestDatabase(t)
	defer teardownDatabase()

	db, err := sql.Open("pgx", dbConn)
	if err != nil {
		t.Fatalf("Failed to open database connection: %v", err)
	}

	defer db.Close()

	runGooseUp(t, db)

	server, teardown := setupTestServer(t)
	defer teardown(server)

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup(db)
			}

			defer executeSQLFile(t, db, "./test_data/cleanup.sql")
			newTestRequest(t, tc, server, db)
		})
	}
}

// Write path and read path were only tested separately: the endpoint test seeds
// audit_events from a fixture, so nothing proved that rows the middleware writes are
// actually readable through the API. This drives real mutations, then reads them back.
func TestAuditLogEndToEnd(t *testing.T) {
	dbConn, teardownDatabase := setupTestDatabase(t)
	defer teardownDatabase()

	db, err := sql.Open("pgx", dbConn)
	if err != nil {
		t.Fatalf("Failed to open database connection: %v", err)
	}

	defer db.Close()

	runGooseUp(t, db)

	server, teardown := setupTestServer(t)
	defer teardown(server)

	defer executeSQLFile(t, db, "./test_data/cleanup.sql")

	mutate := func(method, endpoint, body string, wantStatus int) {
		t.Helper()

		var reader io.Reader
		if body != "" {
			reader = bytes.NewBufferString(body)
		}

		req, err := http.NewRequestWithContext(t.Context(), method, server.URL+endpoint, reader)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer sub-1")
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		defer resp.Body.Close()

		require.Equal(t, wantStatus, resp.StatusCode, "%s %s", method, endpoint)
	}

	mutate("POST", "/games", `{"name":"Turnier","teamSize":1,"tableSize":2,"numberOfRounds":1}`, http.StatusCreated)
	mutate("POST", "/games/1/teams", `{"name":"Team A","players":[{"name":"Anna"}]}`, http.StatusCreated)
	mutate("PUT", "/games/1/teams/1", `{"name":"Die Knobelkoenige"}`, http.StatusOK)
	mutate("PUT", "/games/1", `{"name":"Finale","teamSize":1,"tableSize":2,"numberOfRounds":1}`, http.StatusOK)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+"/games/1/audit", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer sub-1")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var response struct {
		Events []struct {
			ID        int64  `json:"id"`
			Timestamp string `json:"timestamp"`
			RequestID string `json:"requestID"`
			Actor     struct {
				Sub   string `json:"sub"`
				Email string `json:"email"`
			} `json:"actor"`
			Action   string `json:"action"`
			Entity   string `json:"entity"`
			EntityID string `json:"entityID"`
			Changes  []struct {
				Field string  `json:"field"`
				From  *string `json:"from"`
				To    *string `json:"to"`
			} `json:"changes"`
		} `json:"events"`
	}

	require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))
	require.NotEmpty(t, response.Events, "the middleware wrote rows that the endpoint cannot read")

	for i, event := range response.Events {
		assert.Equal(t, "sub-1", event.Actor.Sub)
		assert.Equal(t, "sub-1@example.org", event.Actor.Email)
		assert.NotEmpty(t, event.RequestID)
		assert.NotEmpty(t, event.Timestamp)

		if i > 0 {
			assert.Less(t, event.ID, response.Events[i-1].ID, "events must come back newest first")
		}
	}

	// The four requests are distinguishable by their request ids, and the last two
	// renames must be present with both sides of the change.
	requestIDs := map[string]struct{}{}
	for _, event := range response.Events {
		requestIDs[event.RequestID] = struct{}{}
	}

	assert.Len(t, requestIDs, 4, "each request must group under its own request id")

	renames := map[string]string{}

	for _, event := range response.Events {
		if event.Action != "update" {
			continue
		}

		for _, change := range event.Changes {
			if change.Field == "name" && change.From != nil && change.To != nil {
				renames[*change.From] = *change.To
			}
		}
	}

	assert.Equal(t, "Die Knobelkoenige", renames["Team A"], "the team rename must be readable end to end")
	assert.Equal(t, "Finale", renames["Turnier"], "the game rename must be readable end to end")
}
