-- Deploy flux:0006_add_scraper_role to pg

BEGIN;

CREATE USER flux_scraper WITH PASSWORD :'flux_scraper_password';

GRANT USAGE ON SCHEMA flux TO flux_scraper;

GRANT INSERT ON flux.market_data TO flux_scraper;
GRANT SELECT ON flux.market_data TO flux_scraper; -- in theory this isn't neccesary but it seems required for insert from the POSTGREST Supabase client library

GRANT flux_scraper to authenticator;

COMMIT;
