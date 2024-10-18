-- +goose Up
alter table teams
 add column game_id bigint
        constraint fk_teams_game
            references games(id)
            on delete cascade;

-- +goose Down
alter table teams
 drop column game_id cascade;
