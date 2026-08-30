-- +goose Up

CREATE TABLE audit_events
(
    id bigserial PRIMARY KEY,
    game_id integer NOT NULL,
    table_name text NOT NULL,
    row_id text NOT NULL,
    -- noqa: disable=RF04
    action text NOT NULL,
    actor_sub text NOT NULL,
    actor_email text NOT NULL,
    old_row jsonb,
    new_row jsonb,
    created_at timestamp with time zone NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_events_game_id_id ON audit_events (game_id, id);

-- +goose StatementBegin
CREATE FUNCTION AUDIT_ROW() RETURNS trigger AS
$$
DECLARE
    old_data         jsonb := CASE WHEN TG_OP = 'INSERT' THEN NULL ELSE to_jsonb(OLD) END;
    new_data         jsonb := CASE WHEN TG_OP = 'DELETE' THEN NULL ELSE to_jsonb(NEW) END;
    row_data         jsonb := COALESCE(new_data, old_data);
    resolved_game_id integer;
BEGIN

    resolved_game_id := CASE TG_TABLE_NAME
                            WHEN 'games' THEN (row_data ->> 'id')::integer
                            WHEN 'game_owners' THEN (row_data ->> 'game_id')::integer
                            WHEN 'teams' THEN (row_data ->> 'game_id')::integer
                            WHEN 'players' THEN (SELECT game_id FROM teams WHERE id = (row_data ->> 'team_id')::integer)
                            WHEN 'scores' THEN (SELECT r.game_id
                                                FROM game_tables gt
                                                         JOIN rounds r ON r.id = gt.round_id
                                                WHERE gt.id = (row_data ->> 'table_id')::integer)
        END;

    -- A table with no CASE branch resolves to NULL, which would make its events invisible to
    -- the read endpoint and, on delete, silently discarded by the guard below. On insert and
    -- update every audited table reaches its game through a NOT NULL foreign key, so NULL
    -- here can only mean someone added a trigger without adding a branch.
    IF TG_OP <> 'DELETE' AND resolved_game_id IS NULL THEN
        RAISE EXCEPTION 'audit_row: no game_id resolution for table %', TG_TABLE_NAME;
    END IF;

    -- Cascade suppression: one event per user action, not one per cascaded row. A child
    -- deleted by ON DELETE CASCADE can no longer reach its game because a row in its parent
    -- chain is already gone, so deleting a game records only the game, and deleting a team
    -- records only the team. pg_trigger_depth() cannot make this distinction: RI cascades
    -- run at depth 1, exactly like a direct delete.
    IF TG_OP = 'DELETE' AND TG_TABLE_NAME <> 'games'
        AND NOT EXISTS (SELECT 1 FROM games WHERE id = resolved_game_id) THEN
        RETURN NULL;
    END IF;

    -- GORM's Save always emits an UPDATE and always bumps updated_at, and Postgres fires
    -- this trigger even when no column value differs. Without this guard, re-saving an
    -- unchanged form writes an event indistinguishable from a real change.
    IF TG_OP = 'UPDATE' AND old_data - 'updated_at' = new_data - 'updated_at' THEN
        RETURN NULL;
    END IF;

    INSERT INTO audit_events (game_id, table_name, row_id, action, actor_sub, actor_email, old_row, new_row)
    VALUES (resolved_game_id,
            TG_TABLE_NAME,
            COALESCE(row_data ->> 'id', row_data ->> 'owner_sub'),
            lower(TG_OP),
            COALESCE(NULLIF(current_setting('app.actor_sub', true), ''), 'system'),
            COALESCE(NULLIF(current_setting('app.actor_email', true), ''), ''),
            old_data,
            new_data);

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER audit
AFTER INSERT OR UPDATE OR DELETE
ON games
FOR EACH ROW
EXECUTE FUNCTION AUDIT_ROW();

CREATE TRIGGER audit
AFTER INSERT OR UPDATE OR DELETE
ON game_owners
FOR EACH ROW
EXECUTE FUNCTION AUDIT_ROW();

CREATE TRIGGER audit
AFTER INSERT OR UPDATE OR DELETE
ON teams
FOR EACH ROW
EXECUTE FUNCTION AUDIT_ROW();

CREATE TRIGGER audit
AFTER INSERT OR UPDATE OR DELETE
ON players
FOR EACH ROW
EXECUTE FUNCTION AUDIT_ROW();

CREATE TRIGGER audit
AFTER INSERT OR UPDATE OR DELETE
ON scores
FOR EACH ROW
EXECUTE FUNCTION AUDIT_ROW();
