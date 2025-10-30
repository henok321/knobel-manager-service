INSERT INTO games (
    id, game_name, team_size, table_size, number_of_rounds, status
)
VALUES (1, 'Game 1', 4, 4, 2, 'setup');

INSERT INTO game_owners (game_id, owner_sub)
VALUES (1, 'sub-1');

INSERT INTO teams (id, team_name, game_id)
VALUES (1, 'Team 1', 1),
(2, 'Team 2', 1),
(3, 'Team 3', 1),
(4, 'Team 4', 1);

INSERT INTO players (id, player_name, team_id)
VALUES (1, 'Player 1', 1),
(2, 'Player 2', 1),
(3, 'Player 3', 1),
(4, 'Player 4', 1),
(5, 'Player 5', 2),
(6, 'Player 6', 2),
(7, 'Player 7', 2),
(8, 'Player 8', 2),
(9, 'Player 9', 3),
(10, 'Player 10', 3),
(11, 'Player 11', 3),
(12, 'Player 12', 3),
(13, 'Player 13', 4),
(14, 'Player 14', 4),
(15, 'Player 15', 4),
(16, 'Player 16', 4);
