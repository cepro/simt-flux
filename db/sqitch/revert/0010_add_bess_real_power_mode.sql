-- Revert flux:add_bess_real_power_mode from pg

BEGIN;

ALTER TABLE flux.mg_bess_readings
DROP COLUMN real_power_mode;

COMMIT;
