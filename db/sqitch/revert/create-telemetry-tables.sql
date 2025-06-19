-- Revert flux:create-telemetry-tables from pg

BEGIN;

DROP TABLE flux.mg_meter_readings;
DROP TABLE flux.mg_bess_readings;
DROP TABLE flux.mg_device_registry;

COMMIT;
