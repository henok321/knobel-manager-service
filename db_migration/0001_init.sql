-- +goose Up

CREATE TYPE game_status AS ENUM ('setup', 'in_progress', 'completed');

CREATE TABLE games
(
    id serial PRIMARY KEY,
    -- noqa: disable=RF04
    name varchar(255) NOT NULL,
    team_size integer NOT NULL,
    table_size integer NOT NULL,
    number_of_rounds integer NOT NULL,
    status game_status NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    deleted_at timestamp with time zone,
    UNIQUE (id, deleted_at)
);

CREATE INDEX idx_games_deleted_at ON games (deleted_at);

CREATE TABLE game_owners
(
    game_id integer NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    owner_sub varchar(255) NOT NULL,
    PRIMARY KEY (game_id, owner_sub)
);

CREATE INDEX idx_game_owners_owner_sub ON game_owners (owner_sub);

CREATE TABLE teams
(
    id serial PRIMARY KEY,
    -- noqa: disable=RF04
    name varchar(255) NOT NULL,
    game_id integer NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    CONSTRAINT fk_team_game FOREIGN KEY (game_id) REFERENCES games (
        id
    ) ON DELETE CASCADE
);

CREATE INDEX idx_teams_game_id ON teams (game_id);

CREATE TABLE players
(
    id serial PRIMARY KEY,
    -- noqa: disable=RF04
    name varchar(255) NOT NULL,
    team_id integer NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    CONSTRAINT fk_player_team FOREIGN KEY (team_id) REFERENCES teams (
        id
    ) ON DELETE CASCADE
);

CREATE INDEX idx_players_team_id ON players (team_id);

CREATE TABLE rounds
(
    id serial PRIMARY KEY,
    round_number integer NOT NULL,
    game_id integer NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    status varchar(50) NOT NULL,
    CONSTRAINT unique_game_round UNIQUE (game_id, round_number)
);

CREATE INDEX idx_rounds_game_id ON rounds (game_id);

CREATE TABLE game_tables
(
    id serial PRIMARY KEY,
    table_number integer NOT NULL,
    round_id integer NOT NULL REFERENCES rounds (id) ON DELETE CASCADE,
    CONSTRAINT unique_round_table UNIQUE (round_id, table_number)
);

CREATE INDEX idx_game_tables_round_id ON game_tables (round_id);

CREATE TABLE table_players
(
    game_table_id integer NOT NULL REFERENCES game_tables (
        id
    ) ON DELETE CASCADE,
    player_id integer NOT NULL REFERENCES players (id) ON DELETE CASCADE,
    PRIMARY KEY (game_table_id, player_id)
);

CREATE INDEX idx_table_players_game_table_id ON table_players (game_table_id);
CREATE INDEX idx_table_players_player_id ON table_players (player_id);

CREATE TABLE scores
(
    id serial PRIMARY KEY,
    player_id integer NOT NULL REFERENCES players (id) ON DELETE CASCADE,
    table_id integer NOT NULL REFERENCES game_tables (id) ON DELETE CASCADE,
    score integer NOT NULL,
    CONSTRAINT unique_player_table UNIQUE (player_id, table_id)
);

CREATE INDEX idx_scores_player_id ON scores (player_id);
CREATE INDEX idx_scores_table_id ON scores (table_id);
