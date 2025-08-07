-- Deploy flux:create-market-data-tables to pg

BEGIN;

CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;
CREATE EXTENSION IF NOT EXISTS timescaledb_toolkit WITH SCHEMA "extensions";

-- The market_data table holds various types of data about the electricity markets.
-- For example, imbalance volume, price, day-ahead prices, etc.
CREATE TABLE flux.market_data (
    time        TIMESTAMPTZ NOT NULL,
    type        INTEGER NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    value       REAL
);
SELECT create_hypertable('flux.market_data', by_range('time'));

-- The market_data_sources table describes the various types of data in the market_data table
CREATE TABLE flux.market_data_types (
    id          SERIAL PRIMARY KEY NOT NULL,
    name        TEXT NOT NULL
);

COMMIT;
