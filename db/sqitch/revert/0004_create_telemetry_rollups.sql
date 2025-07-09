-- Revert flux:create-telemetry-rollups from pg

BEGIN;

DROP MATERIALIZED VIEW flux.mg_meter_readings_5m_intermediate;
DROP MATERIALIZED VIEW flux.mg_meter_readings_30m_intermediate;

DROP FUNCTION get_meter_readings_5m(TIMESTAMPTZ, TIMESTAMPTZ, UUID[]);
DROP FUNCTION get_meter_readings_30m(TIMESTAMPTZ, TIMESTAMPTZ, UUID[]);

DROP MATERIALIZED VIEW flux.mg_bess_readings_5m;
DROP MATERIALIZED VIEW flux.mg_bess_readings_30m;

COMMIT;
