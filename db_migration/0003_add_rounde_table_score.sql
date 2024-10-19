-- +goose Up
CREATE TABLE rounds (
    id SERIAL PRIMARY KEY,
    round_number INT NOT NULL,
    game_id INT NOT NULL,
    UNIQUE (round_number, game_id),
    FOREIGN KEY (game_id) REFERENCES games (id));

CREATE TABLE tables (
    id SERIAL PRIMARY KEY,
    round_id INT NOT NULL,
    FOREIGN KEY (round_id) REFERENCES rounds (id));

CREATE TABLE player_scores (
    player_id INT NOT NULL,
    round_id INT NOT NULL,
    score INT NOT NULL,
    PRIMARY KEY (player_id, round_id),
    UNIQUE (player_id, round_id),
    FOREIGN KEY (player_id) REFERENCES players (id),
    FOREIGN KEY (round_id) REFERENCES rounds (id));

-- +goose Down
DROP TABLE player_scores CASCADE;
DROP TABLE tables CASCADE;
DROP TABLE rounds CASCADE;