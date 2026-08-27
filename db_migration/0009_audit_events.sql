-- +goose Up

CREATE TYPE audit_action AS ENUM ('create', 'update', 'delete', 'setup');

CREATE TYPE audit_entity AS ENUM ('game', 'owner', 'team', 'player', 'score');

CREATE TABLE audit_events
(
    id bigserial PRIMARY KEY,
    game_id integer NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    request_id varchar(32) NOT NULL,
    actor_sub varchar(255) NOT NULL,
    actor_email varchar(255) NOT NULL,
    -- noqa: disable=RF04
    action audit_action NOT NULL,
    entity audit_entity NOT NULL,
    entity_id varchar(255) NOT NULL,
    changes jsonb NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_events_game_id_id ON audit_events (game_id, id DESC);
