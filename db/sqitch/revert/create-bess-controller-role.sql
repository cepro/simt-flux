-- Revert flux:create-bess-controller-role from pg

BEGIN;

REVOKE INSERT ON flux.mg_bess_readings FROM besscontroller;
REVOKE INSERT ON flux.mg_meter_readings FROM besscontroller; 
REVOKE SELECT ON flux.mg_bess_readings FROM besscontroller;
REVOKE SELECT ON flux.mg_meter_readings FROM besscontroller; 
REVOKE USAGE ON SCHEMA flux FROM besscontroller;
DROP USER besscontroller;

COMMIT;
