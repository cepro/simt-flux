-- Deploy flux:create-telemetry-tables to pg

BEGIN;


CREATE TABLE flux.mg_bess_readings (
    "time" timestamp with time zone not null,
    "device_id" uuid not null,
    "id" uuid not null default gen_random_uuid(),
    "created_at" timestamp with time zone not null default now(),
    "soe" float4 not null,
    "target_power" float4 not null
);

CREATE TABLE flux.mg_meter_readings (
    "time" timestamp with time zone not null,
    "device_id" uuid not null,
    "id" uuid not null default gen_random_uuid(),
    "created_at" timestamp with time zone not null default now(),
    "frequency" float4,
    "voltage_line_average" float4,
    "current_phase_a" float4,
    "current_phase_b" float4,
    "current_phase_c" float4,
    "current_phase_average" float4,
    "power_phase_a_active" float4,
    "power_phase_b_active" float4,
    "power_phase_c_active" float4,
    "power_total_active" float4,
    "power_total_reactive" float4,
    "power_total_apparent" float4,
    "power_factor_total" float4,
    "energy_imported_active" float4,
    "energy_exported_active" float4,
    "energy_imported_phase_a_active" float4,
    "energy_exported_phase_a_active" float4, 
    "energy_imported_phase_b_active" float4,
    "energy_exported_phase_b_active" float4, 
    "energy_imported_phase_c_active" float4,
    "energy_exported_phase_c_active" float4,
    "energy_imported_reactive" float4,
    "energy_exported_reactive" float4,
    "energy_imported_apparent" float4,
    "energy_exported_apparent" float4,
);

CREATE TABLE flux.mg_device_registry (
    "device_id" uuid PRIMARY KEY,
    "site" text,
    "name" text,
    "type" text
);

SELECT create_hypertable('flux.mg_bess_readings', by_range('time'));
SELECT create_hypertable('flux.mg_meter_readings', by_range('time'));

CREATE UNIQUE INDEX mg_bess_readings_deviceid_time_idx on flux.mg_bess_readings (device_id, time);
CREATE UNIQUE INDEX mg_meter_readings_deviceid_time_idx on flux.mg_meter_readings (device_id, time);

COMMIT;
