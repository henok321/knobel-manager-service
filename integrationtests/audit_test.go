package integrationtests

import (
	"database/sql"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// Seed data is inserted with the triggers live, so it leaves audit rows of its own.
// Clearing them after seeding lets a test assert on exactly what its own action wrote.
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
		`INSERT INTO rounds (id, round_number, game_id, status) VALUES (1, 1, 1, 'in_progress')`,
		`INSERT INTO game_tables (id, table_number, round_id) VALUES (1, 1, 1)`,
		`INSERT INTO scores (id, player_id, table_id, score) VALUES (1, 1, 1, 42)`,
	}

	for _, statement := range statements {
		_, err := db.ExecContext(t.Context(), statement)
		require.NoError(t, err)
	}
}

func TestAuditTriggers(t *testing.T) {
	dbConn, teardownDatabase := setupTestDatabase(t)
	defer teardownDatabase()

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

			// What GORM's Save emits when a form is re-submitted unchanged: every
			// column rewritten to its current value, updated_at bumped.
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

			for _, row := range auditRows(t, db) {
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
