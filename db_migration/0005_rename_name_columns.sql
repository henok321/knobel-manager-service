-- +goose Up

-- Rename name columns to avoid PostgreSQL reserved keyword conflicts

ALTER TABLE games
RENAME COLUMN name TO game_name;

ALTER TABLE teams
RENAME COLUMN name TO team_name;

ALTER TABLE players
RENAME COLUMN name TO player_name;
