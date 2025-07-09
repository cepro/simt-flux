-- Revert flux:create-market-data-functions from pg

BEGIN;

DELETE FUNCTION flux.insert_market_data_batch(data flux.market_data_input[]);

DELETE TYPE flux.market_data_input;

COMMIT;
