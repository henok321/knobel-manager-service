INSERT INTO games (id, name)
VALUES (1, 'game 1'),
       (2, 'game 2');

INSERT INTO owners (id, sub)
VALUES (1, 'sub-1'),
       (2, 'sub-2');

INSERT INTO game_owners (game_id, owner_id)
VALUES (1, 1),
       (2, 2);


INSERT INTO teams (id, name,game_id)
VALUES (1, 'team 1',1), (2, 'team 2',1);

INSERT INTO players (id, name, team_id)
VALUES (1, 'player 1', 1);
INSERT INTO players (id, name, team_id)
VALUES (2, 'player 2', 2);