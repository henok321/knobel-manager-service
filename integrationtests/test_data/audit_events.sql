INSERT INTO games (
    id, game_name, team_size, table_size, number_of_rounds, status
)
VALUES (1, 'Game 1', 4, 4, 2, 'setup');

INSERT INTO game_owners (game_id, owner_sub)
VALUES (1, 'sub-1');

INSERT INTO audit_events (
    id,
    game_id,
    request_id,
    actor_sub,
    actor_email,
    -- noqa: disable=RF04
    action,
    entity,
    entity_id,
    changes,
    created_at
)
VALUES
(
    1,
    1,
    'req-aaa',
    'sub-1',
    'owner@example.com',
    'create',
    'game',
    '1',
    '[{"field":"name","from":null,"to":"Game 1"}]',
    '2026-08-27T10:00:00Z'
),
(
    2,
    1,
    'req-bbb',
    'sub-1',
    'owner@example.com',
    'update',
    'team',
    '5',
    '[{"field":"name","from":"Team A","to":"Team B"}]',
    '2026-08-27T11:00:00Z'
);
