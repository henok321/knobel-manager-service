-- +goose Up

-- each user can select one game which will be the active game
CREATE TABLE active_games
(
    owner_sub VARCHAR(255) NOT NULL PRIMARY KEY,
    game_id   INTEGER      NOT NULL REFERENCES games (id) ON DELETE CASCADE
);