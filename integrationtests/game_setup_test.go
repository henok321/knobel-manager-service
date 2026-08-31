package integrationtests

import (
	"bytes"
	"database/sql"
	"io"
	"net/http"
	"sync"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
)

func TestGameSetup(t *testing.T) {
	tests := map[string]testCase{
		"Setup game tables": {
			method:             "POST",
			endpoint:           "/games/1/setup",
			expectedStatusCode: http.StatusNoContent,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_ready.sql")
			},
			assertions: assertTablesAssigned,
		},
		"Try to setup game tables with out permissions": {
			method:             "POST",
			endpoint:           "/games/1/setup",
			expectedStatusCode: http.StatusForbidden,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-2"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_ready.sql")
			},
		},
		"Try to setup game not in setup state": {
			method:             "POST",
			endpoint:           "/games/1/setup",
			expectedStatusCode: http.StatusConflict,
			expectedBody:       `{"error":"Game is not editable"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_ready.sql")
				_, err := db.ExecContext(t.Context(), "UPDATE games SET status = 'in_progress' WHERE id = 1")
				if err != nil {
					t.Fatalf("Failed to update game status: %v", err)
				}
			},
		},
		"Try to setup game with invalid number of teams": {
			method:             "POST",
			endpoint:           "/games/1/setup",
			expectedStatusCode: http.StatusConflict,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			},
		},
		"Reset game setup": {
			method:             "DELETE",
			endpoint:           "/games/1/setup",
			expectedStatusCode: http.StatusNoContent,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_tables.sql")
			},
			assertions: assertNoTablesAssigned,
		},
		"Reset game setup without assigned tables": {
			method:             "DELETE",
			endpoint:           "/games/1/setup",
			expectedStatusCode: http.StatusNoContent,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_ready.sql")
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()
				assertNoTablesAssigned(t, db)
				assert.Equal(t, 8, countRows(t, db, "SELECT COUNT(*) FROM teams WHERE game_id = 1"), "resetting must leave the teams alone")
				assert.Equal(t, 32, countRows(t, db, "SELECT COUNT(*) FROM players"), "resetting must leave the players alone")
			},
		},
		"Reset game setup without permission": {
			method:             "DELETE",
			endpoint:           "/games/1/setup",
			expectedStatusCode: http.StatusForbidden,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-2"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_tables.sql")
			},
			assertions: assertTablesAssigned,
		},
		"Reset game setup not in setup state": {
			method:             "DELETE",
			endpoint:           "/games/1/setup",
			expectedStatusCode: http.StatusConflict,
			expectedBody:       `{"error":"Game is not editable"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_assigned.sql")
			},
			assertions: assertTablesAssigned,
		},
	}

	dbConn := setupTestDatabase(t)

	db, err := sql.Open("pgx", dbConn)
	if err != nil {
		t.Fatalf("Failed to open database connection: %v", err)
	}

	defer db.Close()

	runGooseUp(t, db)

	server := setupTestServer(t)

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

func TestGameSetupMultipleTimes(t *testing.T) {
	dbConn := setupTestDatabase(t)

	db, err := sql.Open("pgx", dbConn)
	if err != nil {
		t.Fatalf("Failed to open database connection: %v", err)
	}
	defer db.Close()

	runGooseUp(t, db)

	server := setupTestServer(t)

	executeSQLFile(t, db, "./test_data/games_setup_ready.sql")
	defer executeSQLFile(t, db, "./test_data/cleanup.sql")

	// Setup is idempotent at the API level: rounds and tables are recreated on every call.
	tc := testCase{
		method:             "POST",
		endpoint:           "/games/1/setup",
		expectedStatusCode: http.StatusNoContent,
		requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
	}

	t.Run("First setup", func(t *testing.T) {
		newTestRequest(t, tc, server, db)
	})

	t.Run("Second setup", func(t *testing.T) {
		newTestRequest(t, tc, server, db)
	})

	t.Run("Third setup", func(t *testing.T) {
		newTestRequest(t, tc, server, db)
	})
}

func TestAddTeamAfterSetup(t *testing.T) {
	dbConn := setupTestDatabase(t)

	db, err := sql.Open("pgx", dbConn)
	if err != nil {
		t.Fatalf("Failed to open database connection: %v", err)
	}
	defer db.Close()

	runGooseUp(t, db)

	server := setupTestServer(t)

	executeSQLFile(t, db, "./test_data/games_setup_ready.sql")
	advanceSequences(t, db)

	defer executeSQLFile(t, db, "./test_data/cleanup.sql")

	authorized := map[string]string{"Authorization": "Bearer sub-1"}

	steps := []struct {
		name string
		tc   testCase
	}{
		{"Assign tables", testCase{
			method:             "POST",
			endpoint:           "/games/1/setup",
			requestHeaders:     authorized,
			expectedStatusCode: http.StatusNoContent,
			assertions:         assertTablesAssigned,
		}},
		{"Adding a team is rejected while tables are assigned", testCase{
			method:             "POST",
			endpoint:           "/games/1/teams",
			requestBody:        `{"name":"Team 9"}`,
			requestHeaders:     authorized,
			expectedStatusCode: http.StatusConflict,
			expectedBody:       `{"error":"Game setup already assigned, reset the setup first"}`,
			assertions:         assertTablesAssigned,
		}},
		{"Reset the setup", testCase{
			method:             "DELETE",
			endpoint:           "/games/1/setup",
			requestHeaders:     authorized,
			expectedStatusCode: http.StatusNoContent,
			assertions:         assertNoTablesAssigned,
		}},
		{"Add the team", testCase{
			method:             "POST",
			endpoint:           "/games/1/teams",
			requestBody:        `{"name":"Team 9","players":[{"name":"Player 33"},{"name":"Player 34"},{"name":"Player 35"},{"name":"Player 36"}]}`,
			requestHeaders:     authorized,
			expectedStatusCode: http.StatusCreated,
		}},
		{"Assign tables again", testCase{
			method:             "POST",
			endpoint:           "/games/1/setup",
			requestHeaders:     authorized,
			expectedStatusCode: http.StatusNoContent,
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()
				assertTablesAssigned(t, db)
				seats := countRows(t, db, `SELECT COUNT(*) FROM table_players tp
					JOIN players p ON p.id = tp.player_id WHERE p.team_id = 9`)
				assert.Equal(t, 8, seats, "every player of the added team must be seated in both rounds")
			},
		}},
	}

	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			newTestRequest(t, step.tc, server, db)
		})
	}
}

func TestConcurrentTeamCreateAndSetup(t *testing.T) {
	dbConn := setupTestDatabase(t)

	db, err := sql.Open("pgx", dbConn)
	if err != nil {
		t.Fatalf("Failed to open database connection: %v", err)
	}
	defer db.Close()

	runGooseUp(t, db)

	server := setupTestServer(t)

	executeSQLFile(t, db, "./test_data/games_setup_ready.sql")
	advanceSequences(t, db)

	defer executeSQLFile(t, db, "./test_data/cleanup.sql")

	post := func(endpoint, body string) int {
		var reader io.Reader
		if body != "" {
			reader = bytes.NewBufferString(body)
		}

		request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, server.URL+endpoint, reader)
		if err != nil {
			t.Errorf("failed to create request: %v", err)
			return 0
		}

		request.Header.Set("Authorization", "Bearer sub-1")
		request.Header.Set("Content-Type", "application/json")

		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Errorf("failed to perform request: %v", err)
			return 0
		}
		defer response.Body.Close()

		return response.StatusCode
	}

	var (
		waitGroup  sync.WaitGroup
		start      = make(chan struct{})
		teamStatus int
	)

	waitGroup.Add(2)

	go func() {
		defer waitGroup.Done()
		<-start
		teamStatus = post("/games/1/teams", `{"name":"Team 9","players":[{"name":"P33"},{"name":"P34"},{"name":"P35"},{"name":"P36"}]}`)
	}()

	go func() {
		defer waitGroup.Done()
		<-start
		post("/games/1/setup", "")
	}()

	close(start)
	waitGroup.Wait()

	seats := countRows(t, db, `SELECT COUNT(*) FROM table_players tp
		JOIN players p ON p.id = tp.player_id WHERE p.team_id = 9`)

	if teamStatus == http.StatusCreated {
		assert.Equal(t, 8, seats, "a team accepted while setup ran must be seated in both rounds")
		return
	}

	assert.Equal(t, http.StatusConflict, teamStatus, "a team refused during setup must be refused with a conflict")
	assert.Equal(t, 8, countRows(t, db, "SELECT COUNT(*) FROM teams WHERE game_id = 1"), "the refused team must not exist")
}
