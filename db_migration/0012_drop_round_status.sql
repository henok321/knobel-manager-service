-- +goose Up

ALTER TABLE rounds
DROP COLUMN status;

-- +goose Down

ALTER TABLE rounds
ADD COLUMN status VARCHAR(50) NOT NULL DEFAULT 'setup';
