-- Deploy flux:add_bess_real_power_mode to pg
-- Adds Tesla Real Power Mode tracking to BESS readings

BEGIN;

ALTER TABLE flux.mg_bess_readings
ADD COLUMN real_power_mode smallint;

COMMENT ON COLUMN flux.mg_bess_readings.real_power_mode IS
'Tesla Real Power Mode: 0=Off, 1=Direct, 2=Site Control, 3=Go to Energy, 4=Opticaster, 5=Scheduler';

COMMIT;
