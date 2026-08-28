-- +goose Up

DROP INDEX idx_audit_events_game_id_id;

DROP TABLE audit_events;

DROP TYPE audit_action;
DROP TYPE audit_entity;
