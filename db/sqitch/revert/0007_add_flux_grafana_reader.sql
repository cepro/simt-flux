-- Revert flux:0007_add_grafana_reader from pg

BEGIN;

DROP USER IF EXISTS flux_grafana_reader;

COMMIT;