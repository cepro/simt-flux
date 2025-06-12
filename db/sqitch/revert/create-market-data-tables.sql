-- Revert flux:create-market-data-tables from pg

BEGIN;

DROP TABLE flux.market_data;
DROP TABLE flux.market_data_types;

COMMIT;
