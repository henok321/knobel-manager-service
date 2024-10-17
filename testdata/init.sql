create table teams
(
    id   bigserial
        primary key,
    name varchar(255)
);

alter table teams
    owner to test;

create table players
(
    id      bigserial
        primary key,
    name    text,
    team_id bigint
        constraint fk_players_team
            references teams
            on delete cascade
);

alter table players
    owner to test;

create table games
(
    id   bigserial
        primary key,
    name text not null
);

alter table games
    owner to test;

create table owners
(
    id  bigserial
        primary key,
    sub text
);

alter table owners
    owner to test;

create table game_owners
(
    game_id  bigint not null
        constraint fk_game_owners_game
            references games
            on update cascade on delete cascade,
    owner_id bigint not null
        constraint fk_game_owners_owner
            references owners
            on update cascade on delete cascade,
    primary key (game_id, owner_id)
);

alter table game_owners
    owner to test;

