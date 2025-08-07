-- Revert flux:create-flux-schema from pg

BEGIN;

DROP SCHEMA IF EXISTS flux;

COMMIT;
