-- Postgres rewrites the row even when no value changes, so the new tuple
-- version lands at the end of the heap and an unordered SELECT returns these
-- rows last. Loaded after a fixture to prove the API orders its collections
-- instead of leaking physical row order.
UPDATE teams SET team_name = team_name
WHERE id = 2;
UPDATE players SET player_name = player_name
WHERE id = 5;
UPDATE game_tables SET table_number = table_number
WHERE id = 2;
UPDATE scores SET score = score
WHERE id = 2;
