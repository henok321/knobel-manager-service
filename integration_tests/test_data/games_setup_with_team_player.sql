INSERT INTO games (id, name, team_size, table_size, number_of_rounds, status)
VALUES (1, 'Game 1', 4, 4, 2, 'setup');

INSERT INTO game_owners (game_id, owner_sub)
VALUES (1, 'sub-1');

INSERT INTO teams (game_id, id, name)
VALUES (1, 1, 'Team 1');

INSERT INTO players (id, name, team_id)
VALUES (1, 'Player 1', 1);