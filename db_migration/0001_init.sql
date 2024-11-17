-- +goose Up

-- Enum type for GameStatus
CREATE TYPE game_status AS ENUM ('setup', 'in_progress', 'completed');

-- Table: games
CREATE TABLE games
(
    id               SERIAL PRIMARY KEY,
    name             VARCHAR(255)             NOT NULL,
    team_size        INTEGER                  NOT NULL,
    table_size       INTEGER                  NOT NULL,
    number_of_rounds INTEGER                  NOT NULL,
    status           game_status              NOT NULL,
    created_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMP WITH TIME ZONE,
    UNIQUE (id, deleted_at)
);

-- Index for soft delete
CREATE INDEX idx_games_deleted_at ON games (deleted_at);

-- Table: game_owners
CREATE TABLE game_owners
(
    game_id   INTEGER      NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    owner_sub VARCHAR(255) NOT NULL,
    PRIMARY KEY (game_id, owner_sub)
);

-- Index on owner_sub
CREATE INDEX idx_game_owners_owner_sub ON game_owners (owner_sub);

-- Table: team
CREATE TABLE teams
(
    id      SERIAL PRIMARY KEY,
    name    VARCHAR(255) NOT NULL,
    game_id INTEGER      NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    CONSTRAINT fk_team_game FOREIGN KEY (game_id) REFERENCES games (id) ON DELETE CASCADE
);

-- Index on game_id
CREATE INDEX idx_teams_game_id ON teams (game_id);

-- Table: players
CREATE TABLE players
(
    id      SERIAL PRIMARY KEY,
    name    VARCHAR(255) NOT NULL,
    team_id INTEGER      NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    CONSTRAINT fk_player_team FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE
);

-- Index on team_id
CREATE INDEX idx_players_team_id ON players (team_id);

-- Table: rounds
CREATE TABLE rounds
(
    id           SERIAL PRIMARY KEY,
    round_number INTEGER     NOT NULL,
    game_id      INTEGER     NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    status       VARCHAR(50) NOT NULL,
    CONSTRAINT unique_game_round UNIQUE (game_id, round_number)
);

-- Index on game_id
CREATE INDEX idx_rounds_game_id ON rounds (game_id);

-- Table: game_tables
CREATE TABLE game_tables
(
    id           SERIAL PRIMARY KEY,
    table_number INTEGER NOT NULL,
    round_id     INTEGER NOT NULL REFERENCES rounds (id) ON DELETE CASCADE,
    CONSTRAINT unique_round_table UNIQUE (round_id, table_number)
);

-- Index on round_id
CREATE INDEX idx_game_tables_round_id ON game_tables (round_id);

-- Table: table_players
CREATE TABLE table_players
(
    game_table_id  INTEGER NOT NULL REFERENCES game_tables (id) ON DELETE CASCADE,
    player_id INTEGER NOT NULL REFERENCES players (id) ON DELETE CASCADE,
    PRIMARY KEY (game_table_id, player_id)
);

-- Indexes on table_players
CREATE INDEX idx_table_players_game_table_id ON table_players (game_table_id);
CREATE INDEX idx_table_players_player_id ON table_players (player_id);

-- Table: scores
CREATE TABLE scores
(
    id        SERIAL PRIMARY KEY,
    player_id INTEGER NOT NULL REFERENCES players (id) ON DELETE CASCADE,
    table_id  INTEGER NOT NULL REFERENCES game_tables (id) ON DELETE CASCADE,
    score     INTEGER NOT NULL,
    CONSTRAINT unique_player_table UNIQUE (player_id, table_id)
);

-- Indexes on scores
CREATE INDEX idx_scores_player_id ON scores (player_id);
CREATE INDEX idx_scores_table_id ON scores (table_id);
