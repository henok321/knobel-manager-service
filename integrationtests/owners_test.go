package integrationtests

import (
	"database/sql"
	"net/http"
	"slices"
	"sync"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
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

				assert.Equal(t, 1, countRows(t, db, "SELECT count(*) FROM game_owners WHERE game_id=1 AND owner_sub='sub-2'"),
					"expected sub-2 to be an owner")
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

				assert.Equal(t, 0, countRows(t, db, "SELECT count(*) FROM game_owners WHERE game_id=1 AND owner_sub='sub-2'"),
					"expected sub-2 to be removed")
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

func TestConcurrentOwnerRemoval(t *testing.T) {
	dbConn := setupTestDatabase(t)

	db, err := sql.Open("pgx", dbConn)
	if err != nil {
		t.Fatalf("Failed to open database connection: %v", err)
	}

	defer db.Close()

	runGooseUp(t, db)

	server := setupTestServer(t)

	executeSQLFile(t, db, "./test_data/games_setup_two_owners.sql")

	defer executeSQLFile(t, db, "./test_data/cleanup.sql")

	remove := func(callerSub, targetSub string) int {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, server.URL+"/games/1/owners/"+targetSub, nil)
		if err != nil {
			t.Errorf("failed to create request: %v", err)
			return 0
		}

		request.Header.Set("Authorization", "Bearer "+callerSub)

		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Errorf("failed to perform request: %v", err)
			return 0
		}
		defer response.Body.Close()

		return response.StatusCode
	}

	var (
		waitGroup sync.WaitGroup
		start     = make(chan struct{})
		statuses  [2]int
	)

	waitGroup.Add(2)

	go func() {
		defer waitGroup.Done()
		<-start

		statuses[0] = remove("sub-1", "sub-2")
	}()

	go func() {
		defer waitGroup.Done()
		<-start

		statuses[1] = remove("sub-2", "sub-1")
	}()

	close(start)
	waitGroup.Wait()

	assert.Equal(t, 1, countRows(t, db, "SELECT COUNT(*) FROM game_owners WHERE game_id = 1"),
		"two owners removing each other must not leave the game ownerless")

	slices.Sort(statuses[:])
	assert.Equal(t, http.StatusOK, statuses[0], "exactly one removal must succeed")
	assert.Contains(t, []int{http.StatusForbidden, http.StatusConflict}, statuses[1],
		"the loser must be refused: 403 once its own sub is gone, 409 if it would strand the last owner")
}
