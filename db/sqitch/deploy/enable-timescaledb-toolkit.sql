-- Deploy flux:enable-timescaledb-toolkit to pg

BEGIN;

-- we need the timescaledb toolkit for the counter_agg function
CREATE EXTENSION IF NOT EXISTS timescaledb_toolkit WITH SCHEMA "extensions";

COMMIT;
