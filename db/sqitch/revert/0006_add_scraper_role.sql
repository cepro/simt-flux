-- Revert flux:0006_add_scraper_role from pg

BEGIN;

REVOKE flux_scraper FROM authenticator;

REVOKE SELECT ON flux.market_data FROM flux_scraper;
REVOKE INSERT ON flux.market_data FROM flux_scraper;

REVOKE USAGE ON SCHEMA flux FROM flux_scraper;

DROP USER flux_scraper;

COMMIT;
