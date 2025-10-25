INSERT INTO games (
    id, game_name, team_size, table_size, number_of_rounds, status
)
VALUES (1, 'Game 1', 4, 4, 2, 'setup');

INSERT INTO game_owners (game_id, owner_sub)
VALUES (1, 'sub-1');

INSERT INTO active_games (game_id, owner_sub)
VALUES (1, 'sub-1');

INSERT INTO teams (id, team_name, game_id)
VALUES (1, 'Team 1', 1);

INSERT INTO players (id, player_name, team_id)
VALUES (1, 'Player 1', 1),
(2, 'Player 2', 1),
(3, 'Player 3', 1),
(4, 'Player 4', 1);
