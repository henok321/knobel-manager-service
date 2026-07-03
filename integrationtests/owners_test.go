package integrationtests

import (
	"database/sql"
	"net/http"
	"testing"

	_ "github.com/lib/pq"
)

func TestOwners(t *testing.T) {
	tests := map[string]testCase{
		"Add owner by email": {
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

				var count int
				if err := db.QueryRowContext(t.Context(), "SELECT count(*) FROM game_owners WHERE game_id=1 AND owner_sub='sub-2'").Scan(&count); err != nil {
					t.Fatalf("query failed: %v", err)
				}

				if count != 1 {
					t.Fatalf("expected sub-2 to be an owner, got %d rows", count)
				}
			},
		},
		"Add owner unknown email": {
			method:             "POST",
			endpoint:           "/games/1/owners",
			requestBody:        `{"email":"ghost@example.org"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusUnprocessableEntity,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			},
		},
		"Add owner already owner": {
			method:             "POST",
			endpoint:           "/games/1/owners",
			requestBody:        `{"email":"sub-1@example.org"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusConflict,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			},
		},
		"Add owner missing email": {
			method:             "POST",
			endpoint:           "/games/1/owners",
			requestBody:        `{}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusBadRequest,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			},
		},
		"Add owner not owner": {
			method:             "POST",
			endpoint:           "/games/1/owners",
			requestBody:        `{"email":"sub-3@example.org"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-2"},
			expectedStatusCode: http.StatusForbidden,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			},
		},
		"Add owner game not found": {
			method:             "POST",
			endpoint:           "/games/2/owners",
			requestBody:        `{"email":"sub-2@example.org"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusNotFound,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			},
		},
		"Remove owner": {
			method:             "DELETE",
			endpoint:           "/games/1/owners/sub-2",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusOK,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_two_owners.sql")
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()

				var count int
				if err := db.QueryRowContext(t.Context(), "SELECT count(*) FROM game_owners WHERE game_id=1 AND owner_sub='sub-2'").Scan(&count); err != nil {
					t.Fatalf("query failed: %v", err)
				}

				if count != 0 {
					t.Fatalf("expected sub-2 to be removed, got %d rows", count)
				}
			},
		},
		"Remove last owner": {
			method:             "DELETE",
			endpoint:           "/games/1/owners/sub-1",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusConflict,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			},
		},
		"Remove owner not present": {
			method:             "DELETE",
			endpoint:           "/games/1/owners/sub-9",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusNotFound,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_two_owners.sql")
			},
		},
		"Remove owner not owner": {
			method:             "DELETE",
			endpoint:           "/games/1/owners/sub-1",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-3"},
			expectedStatusCode: http.StatusForbidden,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_two_owners.sql")
			},
		},
	}

	dbConn, teardownDatabase := setupTestDatabase(t)
	defer teardownDatabase()

	db, err := sql.Open("postgres", dbConn)
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
