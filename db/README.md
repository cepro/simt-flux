
# Database migrations

This directory contains database migrations to support `flux`, for example:
- `mg_meter_readings` table to hold microgrid meter readings.
- `mg_bess_readings` table to hold readings from an on-site BESS.
- `market_data` table to hold arbitrary timeseries data about the power markets.

These tables are expected to be created in a Supabase deployment as the `bess-controller` uses Supabase authentication mechanisms and Postgrest to upload telemetry.

 The Sqitch tool is used to manage database migrations, to check the status of the migrations on a particular deployment:
 - `cd db/sqitch`
 - `sqitch status --target <deployment-target>`


