package integration_tests

import (
	"database/sql"
	"net/http"
	"testing"

	_ "github.com/lib/pq"
)

func teamTestCases(t *testing.T) []testCase {
	return []testCase{
		{
			name:               "Create team",
			method:             "POST",
			endpoint:           "/games/1/teams",
			requestBody:        `{"name":"Team 1"}`,
			requestHeaders:     map[string]string{"Authorization": "sub-1"},
			expectedStatusCode: http.StatusCreated,
			expectedBody:       `{"team": {"id":1,"name":"Team 1", "gameID":1}}`,
			setup: func(db *sql.DB) {
				err := executeSQLFile(t, db, "./test_data/games_setup.sql")
				if err != nil {
					t.Fatalf("Failed to execute SQL file: %v", err)
				}
			},
		},
		{
			name:               "Create team not owner",
			method:             "POST",
			endpoint:           "/games/1/teams",
			requestBody:        `{"name":"Team 1"}`,
			requestHeaders:     map[string]string{"Authorization": "sub-2"},
			expectedStatusCode: http.StatusForbidden,
			setup: func(db *sql.DB) {
				err := executeSQLFile(t, db, "./test_data/games_setup.sql")
				if err != nil {
					t.Fatalf("Failed to execute SQL file: %v", err)
				}
			},
		},
		{
			name:               "Update team",
			method:             "PUT",
			endpoint:           "/games/1/teams/1",
			requestBody:        `{"name":"Team 1 updated"}`,
			requestHeaders:     map[string]string{"Authorization": "sub-1"},
			expectedStatusCode: http.StatusOK,
			expectedBody:       `{"team": {"id":1,"name":"Team 1 updated", "gameID":1}}`,
			setup: func(db *sql.DB) {
				err := executeSQLFile(t, db, "./test_data/games_setup_with_team.sql")
				if err != nil {
					t.Fatalf("Failed to execute SQL file: %v", err)
				}
			},
		},
		{
			name:               "Update team not owner",
			method:             "PUT",
			endpoint:           "/games/1/teams/1",
			requestBody:        `{"name":"Team 1 updated"}`,
			requestHeaders:     map[string]string{"Authorization": "sub-2"},
			expectedStatusCode: http.StatusForbidden,
			setup: func(db *sql.DB) {
				err := executeSQLFile(t, db, "./test_data/games_setup_with_team.sql")
				if err != nil {
					t.Fatalf("Failed to execute SQL file: %v", err)
				}
			},
		},
		{
			name:               "Delete team",
			method:             "DELETE",
			endpoint:           "/games/1/teams/1",
			requestHeaders:     map[string]string{"Authorization": "sub-1"},
			expectedStatusCode: http.StatusNoContent,
			setup: func(db *sql.DB) {
				err := executeSQLFile(t, db, "./test_data/games_setup_with_team.sql")
				if err != nil {
					t.Fatalf("Failed to execute SQL file: %v", err)
				}
			},
		},
		{
			name:               "Delete team not found",
			method:             "DELETE",
			endpoint:           "/games/1/teams/1",
			requestHeaders:     map[string]string{"Authorization": "sub-1"},
			expectedStatusCode: http.StatusNotFound,
			setup: func(db *sql.DB) {
				err := executeSQLFile(t, db, "./test_data/games_setup.sql")
				if err != nil {
					t.Fatalf("Failed to execute SQL file: %v", err)
				}
			},
		},
		{
			name:               "Delete team not owner",
			method:             "DELETE",
			endpoint:           "/games/1/teams/1",
			requestHeaders:     map[string]string{"Authorization": "sub-2"},
			expectedStatusCode: http.StatusForbidden,
			setup: func(db *sql.DB) {
				err := executeSQLFile(t, db, "./test_data/games_setup.sql")
				if err != nil {
					t.Fatalf("Failed to execute SQL file: %v", err)
				}
			},
		},
	}
}

func TestTeams(t *testing.T) {
	RunTestGroup(t, teamTestCases)
}
