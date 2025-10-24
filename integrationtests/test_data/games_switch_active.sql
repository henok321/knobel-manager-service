INSERT INTO games (
    id, game_name, team_size, table_size, number_of_rounds, status
)
VALUES
(1, 'Game 1', 4, 4, 2, 'setup'),
(2, 'Game 2', 4, 4, 3, 'setup');

INSERT INTO game_owners (game_id, owner_sub)
VALUES
(1, 'sub-1'),
(2, 'sub-1');

INSERT INTO active_games (game_id, owner_sub)
VALUES (1, 'sub-1');
