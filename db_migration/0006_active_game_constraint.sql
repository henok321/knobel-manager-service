-- +goose Up

ALTER TABLE active_games
DROP CONSTRAINT active_games_pkey;

ALTER TABLE active_games
ADD PRIMARY KEY (owner_sub, game_id);
