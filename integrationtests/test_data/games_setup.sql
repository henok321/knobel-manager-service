INSERT INTO games (id, name, team_size, table_size, number_of_rounds, status)
VALUES (1, 'Game 1', 4, 4, 2, 'setup');

INSERT INTO game_owners (game_id, owner_sub)
VALUES (1, 'sub-1');

INSERT INTO active_games (game_id, owner_sub)
VALUES (1, 'sub-1');
