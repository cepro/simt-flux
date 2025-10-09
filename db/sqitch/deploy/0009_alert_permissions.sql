-- Deploy flux:0009_alert_permissions to pg

BEGIN;

GRANT USAGE ON SCHEMA flux TO grafanareader;
GRANT SELECT ON TABLE flux.market_data TO grafanareader;

COMMIT;
