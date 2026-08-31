package integrationtests

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/henok321/knobel-manager-service/api/middleware"
	"github.com/henok321/knobel-manager-service/pkg/audit"
	"github.com/henok321/knobel-manager-service/pkg/entity"
)

type auditRow struct {
	GameID     *int
	Entity     string
	RowID      string
	Action     string
	ActorSub   string
	ActorEmail string
	OldRow     *string
	NewRow     *string
}

func auditRows(t *testing.T, db *sql.DB) []auditRow {
	t.Helper()

	rows, err := db.QueryContext(t.Context(), `SELECT game_id, table_name, row_id, action, actor_sub, actor_email,
                                  old_row::text, new_row::text
                           FROM audit_events ORDER BY id`)
	require.NoError(t, err)

	defer rows.Close()

	var out []auditRow

	for rows.Next() {
		var row auditRow

		require.NoError(t, rows.Scan(&row.GameID, &row.Entity, &row.RowID, &row.Action,
			&row.ActorSub, &row.ActorEmail, &row.OldRow, &row.NewRow))

		out = append(out, row)
	}

	require.NoError(t, rows.Err())

	return out
}

// Seeding runs with the triggers live, so its own audit rows must go before a test asserts.
func resetAuditEvents(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.ExecContext(t.Context(), "TRUNCATE TABLE audit_events RESTART IDENTITY")
	require.NoError(t, err)
}

func seedGameWithPlayerAndScore(t *testing.T, db *sql.DB) {
	t.Helper()

	statements := []string{
		`INSERT INTO games (id, game_name, team_size, table_size, number_of_rounds, status)
         VALUES (1, 'Game 1', 4, 4, 2, 'setup')`,
		`INSERT INTO game_owners (game_id, owner_sub) VALUES (1, 'sub-1')`,
		`INSERT INTO teams (id, team_name, game_id) VALUES (1, 'Team 1', 1)`,
		`INSERT INTO players (id, player_name, team_id) VALUES (1, 'Player 1', 1)`,
		`INSERT INTO rounds (id, round_number, game_id) VALUES (1, 1, 1)`,
		`INSERT INTO game_tables (id, table_number, round_id) VALUES (1, 1, 1)`,
		`INSERT INTO scores (id, player_id, table_id, score) VALUES (1, 1, 1, 42)`,
	}

	for _, statement := range statements {
		_, err := db.ExecContext(t.Context(), statement)
		require.NoError(t, err)
	}
}

type auditEvent struct {
	ID         int64          `json:"id"`
	Entity     string         `json:"entity"`
	EntityID   string         `json:"entityID"`
	Action     string         `json:"action"`
	ActorSub   string         `json:"actorSub"`
	ActorEmail string         `json:"actorEmail"`
	Old        map[string]any `json:"old"`
	New        map[string]any `json:"new"`
}

// Fails when an audited table gains a column: that column reaches every owner with no code change.
func assertPublishedColumns(t *testing.T, events []auditEvent, entity string, expected []string) {
	t.Helper()

	for _, event := range events {
		if event.Entity != entity || event.New == nil {
			continue
		}

		keys := make([]string, 0, len(event.New))
		for key := range event.New {
			keys = append(keys, key)
		}

		assert.ElementsMatch(t, expected, keys, "published column set for %s changed", entity)

		return
	}

	t.Fatalf("no %s event with a new row found; the pin asserts nothing", entity)
}

func readAuditLog(t *testing.T, server *httptest.Server, gameID int, sub string, expectedStatus int) []auditEvent {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet,
		fmt.Sprintf("%s/games/%d/audit", server.URL, gameID), nil)
	require.NoError(t, err)

	request.Header.Set("Authorization", "Bearer "+sub)

	resp, err := http.DefaultClient.Do(request)
	require.NoError(t, err)

	defer resp.Body.Close()

	require.Equal(t, expectedStatus, resp.StatusCode)

	if expectedStatus != http.StatusOK {
		return nil
	}

	response := struct {
		Events []auditEvent `json:"events"`
	}{}

	require.NoError(t, json.NewDecoder(resp.Body).Decode(&response))

	return response.Events
}

func TestAuditTriggers(t *testing.T) {
	dbConn := setupTestDatabase(t)

	db, err := sql.Open("pgx", dbConn)
	require.NoError(t, err)

	defer db.Close()

	runGooseUp(t, db)

	tests := map[string]func(t *testing.T, db *sql.DB){
		"insert records one row with the system actor": func(t *testing.T, db *sql.DB) {
			t.Helper()

			_, err := db.ExecContext(t.Context(), `INSERT INTO games (id, game_name, team_size, table_size, number_of_rounds, status)
                               VALUES (1, 'Game 1', 4, 4, 2, 'setup')`)
			require.NoError(t, err)

			rows := auditRows(t, db)
			require.Len(t, rows, 1)
			assert.Equal(t, "games", rows[0].Entity)
			assert.Equal(t, "insert", rows[0].Action)
			assert.Equal(t, "1", rows[0].RowID)
			assert.Equal(t, "system", rows[0].ActorSub)
			require.NotNil(t, rows[0].GameID)
			assert.Equal(t, 1, *rows[0].GameID)
			assert.Nil(t, rows[0].OldRow)
			require.NotNil(t, rows[0].NewRow)
			assert.Contains(t, *rows[0].NewRow, "Game 1")
		},
		"update records before and after": func(t *testing.T, db *sql.DB) {
			t.Helper()

			seedGameWithPlayerAndScore(t, db)
			resetAuditEvents(t, db)

			_, err := db.ExecContext(t.Context(), `UPDATE games SET game_name = 'Game 1 updated' WHERE id = 1`)
			require.NoError(t, err)

			rows := auditRows(t, db)
			require.Len(t, rows, 1)
			assert.Equal(t, "update", rows[0].Action)
			require.NotNil(t, rows[0].OldRow)
			require.NotNil(t, rows[0].NewRow)
			assert.Contains(t, *rows[0].OldRow, "Game 1")
			assert.NotContains(t, *rows[0].OldRow, "Game 1 updated")
			assert.Contains(t, *rows[0].NewRow, "Game 1 updated")
		},
		"unchanged update writes nothing": func(t *testing.T, db *sql.DB) {
			t.Helper()

			seedGameWithPlayerAndScore(t, db)
			resetAuditEvents(t, db)

			// What GORM's Save emits for an unchanged form: every column rewritten, updated_at bumped.
			_, err := db.ExecContext(t.Context(), `UPDATE games
                               SET game_name = game_name, status = status, updated_at = NOW()
                               WHERE id = 1`)
			require.NoError(t, err)

			assert.Empty(t, auditRows(t, db))
		},
		"game_id resolves through parent tables": func(t *testing.T, db *sql.DB) {
			t.Helper()

			seedGameWithPlayerAndScore(t, db)

			byEntity := map[string]auditRow{}
			for _, row := range auditRows(t, db) {
				byEntity[row.Entity] = row
			}

			require.Contains(t, byEntity, "players")
			require.NotNil(t, byEntity["players"].GameID)
			assert.Equal(t, 1, *byEntity["players"].GameID)

			require.Contains(t, byEntity, "scores")
			require.NotNil(t, byEntity["scores"].GameID)
			assert.Equal(t, 1, *byEntity["scores"].GameID)
		},
		"derived tables are not audited": func(t *testing.T, db *sql.DB) {
			t.Helper()

			seedGameWithPlayerAndScore(t, db)

			_, err := db.ExecContext(t.Context(), `INSERT INTO table_players (game_table_id, player_id) VALUES (1, 1)`)
			require.NoError(t, err)

			rows := auditRows(t, db)
			require.NotEmpty(t, rows, "the seed writes audited rows; an empty result would pass this vacuously")

			for _, row := range rows {
				assert.NotContains(t, []string{"rounds", "game_tables", "table_players"}, row.Entity)
			}
		},
		"direct child delete is recorded": func(t *testing.T, db *sql.DB) {
			t.Helper()

			seedGameWithPlayerAndScore(t, db)
			resetAuditEvents(t, db)

			_, err := db.ExecContext(t.Context(), `DELETE FROM scores WHERE id = 1`)
			require.NoError(t, err)

			rows := auditRows(t, db)
			require.Len(t, rows, 1)
			assert.Equal(t, "scores", rows[0].Entity)
			assert.Equal(t, "delete", rows[0].Action)
			require.NotNil(t, rows[0].GameID)
			assert.Equal(t, 1, *rows[0].GameID)
			require.NotNil(t, rows[0].OldRow)
			assert.Nil(t, rows[0].NewRow)
		},
		"cascade delete is suppressed": func(t *testing.T, db *sql.DB) {
			t.Helper()

			seedGameWithPlayerAndScore(t, db)
			resetAuditEvents(t, db)

			_, err := db.ExecContext(t.Context(), `DELETE FROM games WHERE id = 1`)
			require.NoError(t, err)

			rows := auditRows(t, db)
			require.Len(t, rows, 1, "deleting a game must record one event, not one per cascaded child")
			assert.Equal(t, "games", rows[0].Entity)
			assert.Equal(t, "delete", rows[0].Action)
		},
	}

	for name, run := range tests {
		t.Run(name, func(t *testing.T) {
			defer executeSQLFile(t, db, "./test_data/cleanup.sql")
			run(t, db)
		})
	}
}

func TestAuditActor(t *testing.T) {
	tests := map[string]testCase{
		"create game records the game and its owner": {
			method:             http.MethodPost,
			endpoint:           "/games",
			requestBody:        `{"name":"Game 1","numberOfRounds":2, "teamSize":4, "tableSize":4}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusCreated,
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()

				rows := auditRows(t, db)
				require.Len(t, rows, 2)

				byEntity := map[string]auditRow{}
				for _, row := range rows {
					byEntity[row.Entity] = row
				}

				require.Contains(t, byEntity, "games")
				assert.Equal(t, "insert", byEntity["games"].Action)
				assert.Equal(t, "sub-1", byEntity["games"].ActorSub)

				require.Contains(t, byEntity, "game_owners")
				assert.Equal(t, "sub-1", byEntity["game_owners"].RowID)
				assert.Equal(t, "sub-1", byEntity["game_owners"].ActorSub)
				require.NotNil(t, byEntity["game_owners"].GameID)
				assert.Equal(t, 1, *byEntity["game_owners"].GameID)
			},
		},
		"create team records the calling user": {
			method:             http.MethodPost,
			endpoint:           "/games/1/teams",
			requestBody:        `{"name":"Team 1"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusCreated,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
				resetAuditEvents(t, db)
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()

				rows := auditRows(t, db)
				require.Len(t, rows, 1)
				assert.Equal(t, "teams", rows[0].Entity)
				assert.Equal(t, "insert", rows[0].Action)
				assert.Equal(t, "sub-1", rows[0].ActorSub)
				assert.Equal(t, "sub-1@example.org", rows[0].ActorEmail)
				require.NotNil(t, rows[0].GameID)
				assert.Equal(t, 1, *rows[0].GameID)
			},
		},
		"update game records the calling user": {
			method:             http.MethodPut,
			endpoint:           "/games/1",
			requestBody:        `{"name":"Game 1 updated","numberOfRounds":2, "teamSize":4, "tableSize":4, "status":"setup"}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusOK,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
				resetAuditEvents(t, db)
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()

				var updates []auditRow

				for _, row := range auditRows(t, db) {
					if row.Entity == "games" && row.Action == "update" {
						updates = append(updates, row)
					}
				}

				require.Len(t, updates, 1)
				assert.Equal(t, "sub-1", updates[0].ActorSub)
				require.NotNil(t, updates[0].NewRow)
				assert.Contains(t, *updates[0].NewRow, "Game 1 updated")
			},
		},
		"delete team records one event, not one per cascaded player": {
			method:             http.MethodDelete,
			endpoint:           "/games/1/teams/1",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusNoContent,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_team_player.sql")
				resetAuditEvents(t, db)
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()

				rows := auditRows(t, db)
				require.Len(t, rows, 1,
					"one event per user action: the cascaded player deletion is deliberately not recorded")
				assert.Equal(t, "teams", rows[0].Entity)
				assert.Equal(t, "delete", rows[0].Action)
				assert.Equal(t, "sub-1", rows[0].ActorSub)
			},
		},
		"delete game records one event attributed to the caller": {
			method:             http.MethodDelete,
			endpoint:           "/games/1",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusNoContent,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup_with_team_player.sql")
				resetAuditEvents(t, db)
			},
			assertions: func(t *testing.T, db *sql.DB) {
				t.Helper()

				rows := auditRows(t, db)
				require.Len(t, rows, 1)
				assert.Equal(t, "games", rows[0].Entity)
				assert.Equal(t, "delete", rows[0].Action)
				assert.Equal(t, "sub-1", rows[0].ActorSub)
			},
		},
	}

	dbConn := setupTestDatabase(t)

	db, err := sql.Open("pgx", dbConn)
	require.NoError(t, err)

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

	// The only test driving GORM's Save rather than hand-written SQL, and the reason the no-op guard exists.
	t.Run("score writes are attributed, and resubmitting the same scores records nothing", func(t *testing.T) {
		executeSQLFile(t, db, "./test_data/games_setup_assigned.sql")
		resetAuditEvents(t, db)

		defer executeSQLFile(t, db, "./test_data/cleanup.sql")

		submitScores := testCase{
			method:             http.MethodPut,
			endpoint:           "/games/1/rounds/1/tables/1/scores",
			requestBody:        `{"scores": [{"playerID":1,"score":6},{"playerID":5,"score":3},{"playerID":9,"score":2},{"playerID":13,"score":1}]}`,
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusOK,
		}

		newTestRequest(t, submitScores, server, db)

		rows := auditRows(t, db)
		require.Len(t, rows, 4, "one event per score in the submission")

		for _, row := range rows {
			assert.Equal(t, "scores", row.Entity)
			assert.Equal(t, "sub-1", row.ActorSub)
			assert.Equal(t, "sub-1@example.org", row.ActorEmail)
			require.NotNil(t, row.GameID)
			assert.Equal(t, 1, *row.GameID)
		}

		resetAuditEvents(t, db)

		newTestRequest(t, submitScores, server, db)

		assert.Empty(t, auditRows(t, db),
			"resubmitting identical scores must record nothing: GORM's Save emits an UPDATE "+
				"and bumps updated_at regardless, so only the trigger guard prevents a phantom event")
	})

	t.Run("a deleted game's trail stays readable to its former owner", func(t *testing.T) {
		executeSQLFile(t, db, "./test_data/games_setup.sql")

		defer executeSQLFile(t, db, "./test_data/cleanup.sql")

		newTestRequest(t, testCase{
			method:             http.MethodDelete,
			endpoint:           "/games/1",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusNoContent,
		}, server, db)

		// Why audit_events.game_id has no foreign key: the trail outlives the game it describes.
		events := readAuditLog(t, server, 1, "sub-1", http.StatusOK)
		require.NotEmpty(t, events)

		var deletions int

		for _, event := range events {
			if event.Entity == "games" && event.Action == "delete" {
				deletions++

				assert.Equal(t, "sub-1", event.ActorSub)
			}
		}

		assert.Equal(t, 1, deletions, "the deletion of the game must be readable after it is gone")
	})

	t.Run("a deleted game's trail is not readable to anyone else", func(t *testing.T) {
		executeSQLFile(t, db, "./test_data/games_setup.sql")

		defer executeSQLFile(t, db, "./test_data/cleanup.sql")

		newTestRequest(t, testCase{
			method:             http.MethodDelete,
			endpoint:           "/games/1",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusNoContent,
		}, server, db)

		// 404 rather than 403: a non-owner learns nothing about whether the game existed.
		readAuditLog(t, server, 1, "sub-2", http.StatusNotFound)
	})

	t.Run("a revoked owner cannot read the trail after the game is deleted", func(t *testing.T) {
		executeSQLFile(t, db, "./test_data/games_setup_two_owners.sql")

		defer executeSQLFile(t, db, "./test_data/cleanup.sql")

		newTestRequest(t, testCase{
			method:             http.MethodDelete,
			endpoint:           "/games/1/owners/sub-2",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusOK,
		}, server, db)

		// sub-2 is out while the game lives.
		readAuditLog(t, server, 1, "sub-2", http.StatusForbidden)

		newTestRequest(t, testCase{
			method:             http.MethodDelete,
			endpoint:           "/games/1",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusNoContent,
		}, server, db)

		// Deleting the game must not hand access back: matching any game_owners event and not the latest would.
		readAuditLog(t, server, 1, "sub-2", http.StatusNotFound)

		require.NotEmpty(t, readAuditLog(t, server, 1, "sub-1", http.StatusOK),
			"the remaining owner must still be able to read it")
	})

	t.Run("setup re-run records the scores it destroys", func(t *testing.T) {
		executeSQLFile(t, db, "./test_data/games_setup_assigned_with_scores.sql")

		// The seed leaves the game in progress; setup is only reachable from the setup state.
		_, err := db.ExecContext(t.Context(), `UPDATE games SET status = 'setup' WHERE id = 1`)
		require.NoError(t, err)

		resetAuditEvents(t, db)

		defer executeSQLFile(t, db, "./test_data/cleanup.sql")

		newTestRequest(t, testCase{
			method:             http.MethodPost,
			endpoint:           "/games/1/setup",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusNoContent,
		}, server, db)

		var scoreDeletes int

		for _, row := range auditRows(t, db) {
			if row.Entity == "scores" && row.Action == "delete" {
				scoreDeletes++

				assert.Equal(t, "sub-1", row.ActorSub)
			}
		}

		// ResetGameTables deletes scores before their parents, so they resolve a game and escape cascade suppression.
		assert.Equal(t, 4, scoreDeletes, "every wiped score must be recorded and attributed")
	})

	t.Run("actor does not persist across requests", func(t *testing.T) {
		executeSQLFile(t, db, "./test_data/games_setup_two_owners.sql")
		resetAuditEvents(t, db)

		defer executeSQLFile(t, db, "./test_data/cleanup.sql")

		for _, sub := range []string{"sub-1", "sub-2"} {
			newTestRequest(t, testCase{
				method:             "POST",
				endpoint:           "/games/1/teams",
				requestBody:        `{"name":"Team ` + sub + `"}`,
				requestHeaders:     map[string]string{"Authorization": "Bearer " + sub},
				expectedStatusCode: http.StatusCreated,
			}, server, db)
		}

		rows := auditRows(t, db)
		require.Len(t, rows, 2)
		assert.Equal(t, "sub-1", rows[0].ActorSub)
		assert.Equal(t, "sub-2", rows[1].ActorSub,
			"the second request must not inherit the first actor from the connection pool")
	})
}

func TestAuditLogEndpoint(t *testing.T) {
	tests := map[string]testCase{
		"read audit log not owner": {
			method:             http.MethodGet,
			endpoint:           "/games/1/audit",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-2"},
			expectedStatusCode: http.StatusForbidden,
			setup: func(db *sql.DB) {
				executeSQLFile(t, db, "./test_data/games_setup.sql")
			},
		},
		"read audit log game not found": {
			method:             http.MethodGet,
			endpoint:           "/games/999/audit",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusNotFound,
		},
		"read audit log invalid gameID": {
			method:             http.MethodGet,
			endpoint:           "/games/invalid/audit",
			requestHeaders:     map[string]string{"Authorization": "Bearer sub-1"},
			expectedStatusCode: http.StatusBadRequest,
		},
	}

	dbConn := setupTestDatabase(t)

	db, err := sql.Open("pgx", dbConn)
	require.NoError(t, err)

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

	// Cannot use the shared testCase harness: its assertions get no handle on the server, only a status code.
	t.Run("read audit log newest first", func(t *testing.T) {
		executeSQLFile(t, db, "./test_data/games_setup.sql")
		resetAuditEvents(t, db)

		defer executeSQLFile(t, db, "./test_data/cleanup.sql")

		_, err := db.ExecContext(t.Context(),
			`INSERT INTO teams (id, team_name, game_id) VALUES (1, 'Team 1', 1)`)
		require.NoError(t, err)

		_, err = db.ExecContext(t.Context(),
			`UPDATE games SET game_name = 'Game 1 updated' WHERE id = 1`)
		require.NoError(t, err)

		events := readAuditLog(t, server, 1, "sub-1", http.StatusOK)
		require.Len(t, events, 2)

		assert.Equal(t, "games", events[0].Entity)
		assert.Equal(t, "update", events[0].Action)
		assert.Equal(t, "1", events[0].EntityID)
		assert.Equal(t, "system", events[0].ActorSub)
		require.NotNil(t, events[0].New)
		assert.Equal(t, "Game 1 updated", events[0].New["game_name"])
		require.NotNil(t, events[0].Old)
		assert.Equal(t, "Game 1", events[0].Old["game_name"])

		assert.Equal(t, "teams", events[1].Entity)
		assert.Equal(t, "insert", events[1].Action)
		assert.Nil(t, events[1].Old)

		// Whole rows are published, so pinning the key set turns a widened API into a failing test.
		assertPublishedColumns(t, events, "games", []string{
			"id", "game_name", "team_size", "table_size", "number_of_rounds",
			"status", "created_at", "updated_at",
		})
		assertPublishedColumns(t, events, "teams", []string{
			"id", "team_name", "game_id", "created_at", "updated_at",
		})
	})

	t.Run("published columns are pinned for every audited table", func(t *testing.T) {
		executeSQLFile(t, db, "./test_data/games_setup_assigned_with_scores.sql")

		defer executeSQLFile(t, db, "./test_data/cleanup.sql")

		events := readAuditLog(t, server, 1, "sub-1", http.StatusOK)
		require.NotEmpty(t, events)

		assertPublishedColumns(t, events, "game_owners", []string{"game_id", "owner_sub"})
		assertPublishedColumns(t, events, "players", []string{
			"id", "player_name", "team_id", "created_at", "updated_at",
		})
		assertPublishedColumns(t, events, "scores", []string{
			"id", "player_id", "table_id", "score", "created_at", "updated_at",
		})
	})

	t.Run("a game's log excludes another game's events", func(t *testing.T) {
		executeSQLFile(t, db, "./test_data/games_setup.sql")

		defer executeSQLFile(t, db, "./test_data/cleanup.sql")

		_, err := db.ExecContext(t.Context(),
			`INSERT INTO games (id, game_name, team_size, table_size, number_of_rounds, status)
			 VALUES (2, 'Game 2', 4, 4, 2, 'setup')`)
		require.NoError(t, err)

		_, err = db.ExecContext(t.Context(),
			`INSERT INTO game_owners (game_id, owner_sub) VALUES (2, 'sub-1')`)
		require.NoError(t, err)

		gameOne := readAuditLog(t, server, 1, "sub-1", http.StatusOK)
		require.NotEmpty(t, gameOne)

		for _, event := range gameOne {
			require.NotNil(t, event.New, "every event here is an insert or update")
			assert.NotEqual(t, "Game 2", event.New["game_name"],
				"game 1's log must not contain game 2's events")
		}

		gameTwo := readAuditLog(t, server, 2, "sub-1", http.StatusOK)
		require.Len(t, gameTwo, 2, "game 2 has exactly its own insert plus its owner")

		for _, event := range gameTwo {
			assert.NotEqual(t, "Game 1", event.New["game_name"])
		}
	})
}

// Callback-anchor guard: GORM writes a belongs-to association as its own nested create, which an
// anchor on "gorm:begin_transaction" would leave unattributed.
func TestAuditActorCoversAssociations(t *testing.T) {
	dbConn := setupTestDatabase(t)

	db, err := sql.Open("pgx", dbConn)
	require.NoError(t, err)

	defer db.Close()

	runGooseUp(t, db)

	gormDB, err := audit.OpenDatabase(dbConn)
	require.NoError(t, err)

	defer executeSQLFile(t, db, "./test_data/cleanup.sql")

	ctx := middleware.ContextWithUser(t.Context(), &middleware.User{
		Sub:   "sub-1",
		Email: "sub-1@example.org",
	})

	team := entity.Team{
		Name: "Team 1",
		Game: &entity.Game{
			Name: "Game 1", TeamSize: 4, TableSize: 4, NumberOfRounds: 2, Status: entity.StatusSetup,
		},
	}
	require.NoError(t, gormDB.WithContext(ctx).Create(&team).Error)

	byEntity := map[string]auditRow{}
	for _, row := range auditRows(t, db) {
		byEntity[row.Entity] = row
	}

	require.Contains(t, byEntity, "games")
	assert.Equal(t, "sub-1", byEntity["games"].ActorSub,
		"the belongs-to game is written before the parent row; it must still be attributed")

	require.Contains(t, byEntity, "teams")
	assert.Equal(t, "sub-1", byEntity["teams"].ActorSub)
}
