-- +goose Up

ALTER TABLE teams
DROP CONSTRAINT IF EXISTS fk_team_game;

ALTER TABLE players
DROP CONSTRAINT IF EXISTS fk_player_team;
