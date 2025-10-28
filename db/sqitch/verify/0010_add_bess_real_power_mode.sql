-- Verify flux:add_bess_real_power_mode on pg

BEGIN;

SELECT real_power_mode
FROM flux.mg_bess_readings
WHERE FALSE;

ROLLBACK;
