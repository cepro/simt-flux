-- Deploy flux:create-flux-schema to pg

BEGIN;

CREATE SCHEMA IF NOT EXISTS flux;

COMMIT;
