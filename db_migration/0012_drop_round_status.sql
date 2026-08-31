-- +goose Up

ALTER TABLE rounds
DROP COLUMN status;
