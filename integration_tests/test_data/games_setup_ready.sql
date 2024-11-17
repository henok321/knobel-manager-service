INSERT INTO games (id, name, team_size, table_size, number_of_rounds, status)
VALUES (1, 'Game 1', 4, 4, 2, 'setup');

INSERT INTO game_owners (game_id, owner_sub)
VALUES (1, 'sub-1');

INSERT INTO teams (game_id, id, name)
VALUES (1, 1, 'Team 1'),
       (1, 2, 'Team 2'),
       (1, 3, 'Team 3'),
       (1, 4, 'Team 4'),
       (1, 5, 'Team 5'),
       (1, 6, 'Team 6'),
       (1, 7, 'Team 7'),
       (1, 8, 'Team 8');


INSERT INTO players (id, name, team_id)
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
       (16, 'Player 16', 4),
       (17, 'Player 17', 5),
       (18, 'Player 18', 5),
       (19, 'Player 19', 5),
       (20, 'Player 20', 5),
       (21, 'Player 21', 6),
       (22, 'Player 22', 6),
       (23, 'Player 23', 6),
       (24, 'Player 24', 6),
       (25, 'Player 25', 7),
       (26, 'Player 26', 7),
       (27, 'Player 27', 7),
       (28, 'Player 28', 7),
       (29, 'Player 29', 8),
       (30, 'Player 30', 8),
       (31, 'Player 31', 8),
       (32, 'Player 32', 8);
