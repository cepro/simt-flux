-- Deploy flux:create-bess-controller-role to pg

BEGIN;

-- Create a role specifically for the bess controller which accesses this DB via foreign tables
CREATE USER besscontroller WITH PASSWORD :'besscontroller_password';

GRANT USAGE ON SCHEMA flux TO besscontroller;

GRANT INSERT ON flux.mg_bess_readings TO besscontroller;
GRANT INSERT ON flux.mg_meter_readings TO besscontroller;

-- Not sure why SELECT is a required permission, but it doesn't work without it!
GRANT SELECT ON flux.mg_bess_readings TO besscontroller;
GRANT SELECT ON flux.mg_meter_readings TO besscontroller;

GRANT besscontroller to authenticator;

COMMIT;
