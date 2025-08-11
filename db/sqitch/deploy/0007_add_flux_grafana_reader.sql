-- Deploy flux:0007_add_grafana_reader to pg

BEGIN;

CREATE USER flux_grafana_reader WITH PASSWORD :'flux_grafana_reader_password';

GRANT USAGE ON SCHEMA flux TO flux_grafana_reader;

GRANT SELECT ON flux.mg_device_registry TO flux_grafana_reader;
GRANT SELECT ON flux.mg_meter_readings TO flux_grafana_reader;
GRANT SELECT ON flux.mg_bess_readings TO flux_grafana_reader;
GRANT SELECT ON flux.market_data TO flux_grafana_reader;
GRANT SELECT ON flux.market_data_types TO flux_grafana_reader;

GRANT SELECT ON flux.mg_meter_readings_5m_intermediate TO flux_grafana_reader;
GRANT SELECT ON flux.mg_meter_readings_30m_intermediate TO flux_grafana_reader;

ALTER ROLE flux_grafana_reader SET search_path = flux,public;

GRANT USAGE ON SCHEMA public TO flux_grafana_reader;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO flux_grafana_reader; -- for time_bucket etc

COMMIT;
