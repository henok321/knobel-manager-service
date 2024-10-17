INSERT INTO games (id, name)
VALUES (1, 'game 1'),
       (2, 'game 2');

INSERT INTO owners (id, sub)
VALUES (1, 'sub-1'),
       (2, 'sub-2');

INSERT INTO game_owners (game_id, owner_id)
VALUES (1, 1),
       (2, 2);
