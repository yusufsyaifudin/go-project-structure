-- +migrate Up
-- SQL in section 'Up' is executed when this migration is applied.
--
-- This is a placeholder no-op migration kept so the migrations/ mechanism
-- (sql-migrate + go:embed) has at least one file to embed and remains
-- demonstrably functional in this base project template. Replace or remove
-- it once real migrations are added.
SELECT 1;

-- +migrate Down
-- SQL in section 'Down' is executed when this migration is rolled back.
SELECT 1;
