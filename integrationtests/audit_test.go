package integrationtests

import (
	"database/sql"
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
}

func queryAuditEvents(t *testing.T, db *sql.DB) []auditRow {
	t.Helper()

	rows, err := db.QueryContext(t.Context(),
		"SELECT action, entity, entity_id, actor_sub, actor_email, changes FROM audit_events ORDER BY id")
	require.NoError(t, err, "cannot read audit events")

	defer rows.Close()

	events := make([]auditRow, 0)

	for rows.Next() {
		var row auditRow
		require.NoError(t, rows.Scan(&row.action, &row.entity, &row.entityID, &row.actorSub, &row.actorMail, &row.changes))
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
					"changes":[{"field":"name","to":"Game 1"}]
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
