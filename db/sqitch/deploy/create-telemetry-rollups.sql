-- Deploy flux:create-telemetry-rollups to pg

BEGIN;


CREATE MATERIALIZED VIEW flux.meter_readings_5m_intermediate
WITH (timescaledb.continuous) AS
select
    device_id,
    time_bucket('5m', time) as time_b,
    avg(frequency) as frequency_avg,
    avg(voltage_line_average) as voltage_line_average_avg,
    avg(current_phase_a) as current_phase_a_avg,
    avg(current_phase_b) as current_phase_b_avg,
    avg(current_phase_c) as current_phase_c_avg,
    avg(current_phase_average) as current_phase_average_avg,
    avg(power_phase_a_active) as power_phase_a_active_avg,
    avg(power_phase_b_active) as power_phase_b_active_avg,
    avg(power_phase_c_active) as power_phase_c_active_avg,
    avg(power_total_active) as power_total_active_avg,
    avg(power_total_reactive) as power_total_reactive_avg,
    avg(power_total_apparent) as power_total_apparent_avg,
    avg(power_factor_total) as power_factor_total_avg,
    min(energy_imported_active) as energy_imported_active_min,
    min(energy_exported_active) as energy_exported_active_min,
    min(energy_imported_phase_a_active) as energy_imported_phase_a_active_min,
    min(energy_exported_phase_a_active) as energy_exported_phase_a_active_min, 
    min(energy_imported_phase_b_active) as energy_imported_phase_b_active_min,
    min(energy_exported_phase_b_active) as energy_exported_phase_b_active_min, 
    min(energy_imported_phase_c_active) as energy_imported_phase_c_active_min,
    min(energy_exported_phase_c_active) as energy_exported_phase_c_active_min,
    counter_agg(time, energy_imported_active) as energy_imported_active_counter_agg,
    counter_agg(time, energy_exported_active) as energy_exported_active_counter_agg,
    counter_agg(time, energy_imported_phase_a_active) as energy_imported_phase_a_active_counter_agg,
    counter_agg(time, energy_exported_phase_a_active) as energy_exported_phase_a_active_counter_agg,
    counter_agg(time, energy_imported_phase_b_active) as energy_imported_phase_b_active_counter_agg,
    counter_agg(time, energy_exported_phase_b_active) as energy_exported_phase_b_active_counter_agg,
    counter_agg(time, energy_imported_phase_c_active) as energy_imported_phase_c_active_counter_agg,
    counter_agg(time, energy_exported_phase_c_active) as energy_exported_phase_c_active_counter_agg,
    counter_agg(time, energy_imported_reactive) as energy_imported_reactive_counter_agg,
    counter_agg(time, energy_exported_reactive) as energy_exported_reactive_counter_agg,
    counter_agg(time, energy_imported_apparent) as energy_imported_apparent_counter_agg,
    counter_agg(time, energy_exported_apparent) as energy_exported_apparent_counter_agg
FROM flux.meter_readings
GROUP BY device_id, time_b;

CREATE MATERIALIZED VIEW flux.meter_readings_30m_intermediate
WITH (timescaledb.continuous) AS
select
    device_id,
    time_bucket('30m', time) as time_b,
    avg(frequency) as frequency_avg,
    avg(voltage_line_average) as voltage_line_average_avg,
    avg(current_phase_a) as current_phase_a_avg,
    avg(current_phase_b) as current_phase_b_avg,
    avg(current_phase_c) as current_phase_c_avg,
    avg(current_phase_average) as current_phase_average_avg,
    avg(power_phase_a_active) as power_phase_a_active_avg,
    avg(power_phase_b_active) as power_phase_b_active_avg,
    avg(power_phase_c_active) as power_phase_c_active_avg,
    avg(power_total_active) as power_total_active_avg,
    avg(power_total_reactive) as power_total_reactive_avg,
    avg(power_total_apparent) as power_total_apparent_avg,
    avg(power_factor_total) as power_factor_total_avg,
    min(energy_imported_active) as energy_imported_active_min,
    min(energy_exported_active) as energy_exported_active_min,
    min(energy_imported_phase_a_active) as energy_imported_phase_a_active_min,
    min(energy_exported_phase_a_active) as energy_exported_phase_a_active_min, 
    min(energy_imported_phase_b_active) as energy_imported_phase_b_active_min,
    min(energy_exported_phase_b_active) as energy_exported_phase_b_active_min, 
    min(energy_imported_phase_c_active) as energy_imported_phase_c_active_min,
    min(energy_exported_phase_c_active) as energy_exported_phase_c_active_min,
    counter_agg(time, energy_imported_active) as energy_imported_active_counter_agg,
    counter_agg(time, energy_exported_active) as energy_exported_active_counter_agg,
    counter_agg(time, energy_imported_phase_a_active) as energy_imported_phase_a_active_counter_agg,
    counter_agg(time, energy_exported_phase_a_active) as energy_exported_phase_a_active_counter_agg,
    counter_agg(time, energy_imported_phase_b_active) as energy_imported_phase_b_active_counter_agg,
    counter_agg(time, energy_exported_phase_b_active) as energy_exported_phase_b_active_counter_agg,
    counter_agg(time, energy_imported_phase_c_active) as energy_imported_phase_c_active_counter_agg,
    counter_agg(time, energy_exported_phase_c_active) as energy_exported_phase_c_active_counter_agg,
    counter_agg(time, energy_imported_reactive) as energy_imported_reactive_counter_agg,
    counter_agg(time, energy_exported_reactive) as energy_exported_reactive_counter_agg,
    counter_agg(time, energy_imported_apparent) as energy_imported_apparent_counter_agg,
    counter_agg(time, energy_exported_apparent) as energy_exported_apparent_counter_agg
FROM flux.meter_readings
GROUP BY device_id, time_b;


CREATE OR REPLACE FUNCTION get_meter_readings_5m(
    start_time TIMESTAMPTZ DEFAULT now() - INTERVAL '24 hours',
    end_time TIMESTAMPTZ DEFAULT now(),
    device_ids UUID[] DEFAULT NULL
)
RETURNS TABLE(
    time_b TIMESTAMPTZ,
    device_id UUID,
    frequency_avg DOUBLE PRECISION,
    voltage_line_average_avg DOUBLE PRECISION,
    current_phase_a_avg DOUBLE PRECISION,
    current_phase_b_avg DOUBLE PRECISION,
    current_phase_c_avg DOUBLE PRECISION,
    current_phase_average_avg DOUBLE PRECISION,
    power_phase_a_active_avg DOUBLE PRECISION,
    power_phase_b_active_avg DOUBLE PRECISION,
    power_phase_c_active_avg DOUBLE PRECISION,
    power_total_active_avg DOUBLE PRECISION,
    power_total_reactive_avg DOUBLE PRECISION,
    power_total_apparent_avg DOUBLE PRECISION,
    power_factor_total_avg DOUBLE PRECISION,
    energy_imported_active_min REAL,
    energy_exported_active_min REAL,
    energy_imported_phase_a_active_min REAL,
    energy_exported_phase_a_active_min REAL,
    energy_imported_phase_b_active_min REAL,
    energy_exported_phase_b_active_min REAL,
    energy_imported_phase_c_active_min REAL,
    energy_exported_phase_c_active_min REAL,
    energy_imported_active_delta DOUBLE PRECISION,
    energy_exported_active_delta DOUBLE PRECISION,
    energy_imported_phase_a_active_delta DOUBLE PRECISION,
    energy_exported_phase_a_active_delta DOUBLE PRECISION,
    energy_imported_phase_b_active_delta DOUBLE PRECISION,
    energy_exported_phase_b_active_delta DOUBLE PRECISION,
    energy_imported_phase_c_active_delta DOUBLE PRECISION,
    energy_exported_phase_c_active_delta DOUBLE PRECISION
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        t.time_b, 
        t.device_id, 
        t.frequency_avg,
        t.voltage_line_average_avg,
        t.current_phase_a_avg,
        t.current_phase_b_avg,
        t.current_phase_c_avg,
        t.current_phase_average_avg,
        t.power_phase_a_active_avg,
        t.power_phase_b_active_avg,
        t.power_phase_c_active_avg,
        t.power_total_active_avg,
        t.power_total_reactive_avg,
        t.power_total_apparent_avg,
        t.power_factor_total_avg,
        t.energy_imported_active_min,
        t.energy_exported_active_min,
        t.energy_imported_phase_a_active_min,
        t.energy_exported_phase_a_active_min, 
        t.energy_imported_phase_b_active_min,
        t.energy_exported_phase_b_active_min, 
        t.energy_imported_phase_c_active_min,
        t.energy_exported_phase_c_active_min,
        
        CASE WHEN t.energy_imported_active_counter_agg IS NULL THEN
          NULL 
        ELSE
          interpolated_delta(t.energy_imported_active_counter_agg, t.time_b, '5m', 
                             LAG(t.energy_imported_active_counter_agg) OVER ordered_meter, 
                             LEAD(t.energy_imported_active_counter_agg) OVER ordered_meter)
        END AS energy_imported_active_delta,
        
        CASE WHEN t.energy_exported_active_counter_agg IS NULL THEN
          NULL 
        ELSE
          interpolated_delta(t.energy_exported_active_counter_agg, t.time_b, '5m', 
                             LAG(t.energy_exported_active_counter_agg) OVER ordered_meter, 
                             LEAD(t.energy_exported_active_counter_agg) OVER ordered_meter)
        END AS energy_exported_active_delta,
        
        CASE WHEN t.energy_imported_phase_a_active_counter_agg IS NULL THEN
          NULL 
        ELSE
          interpolated_delta(t.energy_imported_phase_a_active_counter_agg, t.time_b, '5m', 
                             LAG(t.energy_imported_phase_a_active_counter_agg) OVER ordered_meter, 
                             LEAD(t.energy_imported_phase_a_active_counter_agg) OVER ordered_meter)
        END AS energy_imported_phase_a_active_delta,
        
        CASE WHEN t.energy_exported_phase_a_active_counter_agg IS NULL THEN
          NULL 
        ELSE
          interpolated_delta(t.energy_exported_phase_a_active_counter_agg, t.time_b, '5m', 
                             LAG(t.energy_exported_phase_a_active_counter_agg) OVER ordered_meter, 
                             LEAD(t.energy_exported_phase_a_active_counter_agg) OVER ordered_meter)
        END AS energy_exported_phase_a_active_delta,
        
        CASE WHEN t.energy_imported_phase_b_active_counter_agg IS NULL THEN
          NULL 
        ELSE
          interpolated_delta(t.energy_imported_phase_b_active_counter_agg, t.time_b, '5m', 
                             LAG(t.energy_imported_phase_b_active_counter_agg) OVER ordered_meter, 
                             LEAD(t.energy_imported_phase_b_active_counter_agg) OVER ordered_meter)
        END AS energy_imported_phase_b_active_delta,
        
        CASE WHEN t.energy_exported_phase_b_active_counter_agg IS NULL THEN
          NULL 
        ELSE
          interpolated_delta(t.energy_exported_phase_b_active_counter_agg, t.time_b, '5m', 
                             LAG(t.energy_exported_phase_b_active_counter_agg) OVER ordered_meter, 
                             LEAD(t.energy_exported_phase_b_active_counter_agg) OVER ordered_meter)
        END AS energy_exported_phase_b_active_delta,
        
        CASE WHEN t.energy_imported_phase_c_active_counter_agg IS NULL THEN
          NULL 
        ELSE
          interpolated_delta(t.energy_imported_phase_c_active_counter_agg, t.time_b, '5m', 
                             LAG(t.energy_imported_phase_c_active_counter_agg) OVER ordered_meter, 
                             LEAD(t.energy_imported_phase_c_active_counter_agg) OVER ordered_meter)
        END AS energy_imported_phase_c_active_delta,
        
        CASE WHEN t.energy_exported_phase_c_active_counter_agg IS NULL THEN
          NULL 
        ELSE
          interpolated_delta(t.energy_exported_phase_c_active_counter_agg, t.time_b, '5m', 
                             LAG(t.energy_exported_phase_c_active_counter_agg) OVER ordered_meter, 
                             LEAD(t.energy_exported_phase_c_active_counter_agg) OVER ordered_meter)
        END AS energy_exported_phase_c_active_delta
        
    FROM flows.mg_meter_readings_5m_intermediate t
    WHERE t.time_b BETWEEN start_time AND end_time
		AND (device_ids IS NULL OR t.device_id = ANY(device_ids))
    WINDOW ordered_meter AS (PARTITION BY t.device_id ORDER BY t.time_b);
END;
$$ LANGUAGE plpgsql;


-- Do the same with the 30m aggregation
CREATE OR REPLACE FUNCTION get_meter_readings_30m(
    start_time TIMESTAMPTZ DEFAULT now() - INTERVAL '24 hours',
    end_time TIMESTAMPTZ DEFAULT now(),
    device_ids UUID[] DEFAULT NULL
)
RETURNS TABLE(
    time_b TIMESTAMPTZ,
    device_id UUID,
    frequency_avg DOUBLE PRECISION,
    voltage_line_average_avg DOUBLE PRECISION,
    current_phase_a_avg DOUBLE PRECISION,
    current_phase_b_avg DOUBLE PRECISION,
    current_phase_c_avg DOUBLE PRECISION,
    current_phase_average_avg DOUBLE PRECISION,
    power_phase_a_active_avg DOUBLE PRECISION,
    power_phase_b_active_avg DOUBLE PRECISION,
    power_phase_c_active_avg DOUBLE PRECISION,
    power_total_active_avg DOUBLE PRECISION,
    power_total_reactive_avg DOUBLE PRECISION,
    power_total_apparent_avg DOUBLE PRECISION,
    power_factor_total_avg DOUBLE PRECISION,
    energy_imported_active_min REAL,
    energy_exported_active_min REAL,
    energy_imported_phase_a_active_min REAL,
    energy_exported_phase_a_active_min REAL,
    energy_imported_phase_b_active_min REAL,
    energy_exported_phase_b_active_min REAL,
    energy_imported_phase_c_active_min REAL,
    energy_exported_phase_c_active_min REAL,
    energy_imported_active_delta DOUBLE PRECISION,
    energy_exported_active_delta DOUBLE PRECISION,
    energy_imported_phase_a_active_delta DOUBLE PRECISION,
    energy_exported_phase_a_active_delta DOUBLE PRECISION,
    energy_imported_phase_b_active_delta DOUBLE PRECISION,
    energy_exported_phase_b_active_delta DOUBLE PRECISION,
    energy_imported_phase_c_active_delta DOUBLE PRECISION,
    energy_exported_phase_c_active_delta DOUBLE PRECISION
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        t.time_b, 
        t.device_id, 
        t.frequency_avg,
        t.voltage_line_average_avg,
        t.current_phase_a_avg,
        t.current_phase_b_avg,
        t.current_phase_c_avg,
        t.current_phase_average_avg,
        t.power_phase_a_active_avg,
        t.power_phase_b_active_avg,
        t.power_phase_c_active_avg,
        t.power_total_active_avg,
        t.power_total_reactive_avg,
        t.power_total_apparent_avg,
        t.power_factor_total_avg,
        t.energy_imported_active_min,
        t.energy_exported_active_min,
        t.energy_imported_phase_a_active_min,
        t.energy_exported_phase_a_active_min, 
        t.energy_imported_phase_b_active_min,
        t.energy_exported_phase_b_active_min, 
        t.energy_imported_phase_c_active_min,
        t.energy_exported_phase_c_active_min,
        
        CASE WHEN t.energy_imported_active_counter_agg IS NULL THEN
          NULL 
        ELSE
          interpolated_delta(t.energy_imported_active_counter_agg, t.time_b, '30m', 
                             LAG(t.energy_imported_active_counter_agg) OVER ordered_meter, 
                             LEAD(t.energy_imported_active_counter_agg) OVER ordered_meter)
        END AS energy_imported_active_delta,
        
        CASE WHEN t.energy_exported_active_counter_agg IS NULL THEN
          NULL 
        ELSE
          interpolated_delta(t.energy_exported_active_counter_agg, t.time_b, '30m', 
                             LAG(t.energy_exported_active_counter_agg) OVER ordered_meter, 
                             LEAD(t.energy_exported_active_counter_agg) OVER ordered_meter)
        END AS energy_exported_active_delta,
        
        CASE WHEN t.energy_imported_phase_a_active_counter_agg IS NULL THEN
          NULL 
        ELSE
          interpolated_delta(t.energy_imported_phase_a_active_counter_agg, t.time_b, '30m', 
                             LAG(t.energy_imported_phase_a_active_counter_agg) OVER ordered_meter, 
                             LEAD(t.energy_imported_phase_a_active_counter_agg) OVER ordered_meter)
        END AS energy_imported_phase_a_active_delta,
        
        CASE WHEN t.energy_exported_phase_a_active_counter_agg IS NULL THEN
          NULL 
        ELSE
          interpolated_delta(t.energy_exported_phase_a_active_counter_agg, t.time_b, '30m', 
                             LAG(t.energy_exported_phase_a_active_counter_agg) OVER ordered_meter, 
                             LEAD(t.energy_exported_phase_a_active_counter_agg) OVER ordered_meter)
        END AS energy_exported_phase_a_active_delta,
        
        CASE WHEN t.energy_imported_phase_b_active_counter_agg IS NULL THEN
          NULL 
        ELSE
          interpolated_delta(t.energy_imported_phase_b_active_counter_agg, t.time_b, '30m', 
                             LAG(t.energy_imported_phase_b_active_counter_agg) OVER ordered_meter, 
                             LEAD(t.energy_imported_phase_b_active_counter_agg) OVER ordered_meter)
        END AS energy_imported_phase_b_active_delta,
        
        CASE WHEN t.energy_exported_phase_b_active_counter_agg IS NULL THEN
          NULL 
        ELSE
          interpolated_delta(t.energy_exported_phase_b_active_counter_agg, t.time_b, '30m', 
                             LAG(t.energy_exported_phase_b_active_counter_agg) OVER ordered_meter, 
                             LEAD(t.energy_exported_phase_b_active_counter_agg) OVER ordered_meter)
        END AS energy_exported_phase_b_active_delta,
        
        CASE WHEN t.energy_imported_phase_c_active_counter_agg IS NULL THEN
          NULL 
        ELSE
          interpolated_delta(t.energy_imported_phase_c_active_counter_agg, t.time_b, '30m', 
                             LAG(t.energy_imported_phase_c_active_counter_agg) OVER ordered_meter, 
                             LEAD(t.energy_imported_phase_c_active_counter_agg) OVER ordered_meter)
        END AS energy_imported_phase_c_active_delta,
        
        CASE WHEN t.energy_exported_phase_c_active_counter_agg IS NULL THEN
          NULL 
        ELSE
          interpolated_delta(t.energy_exported_phase_c_active_counter_agg, t.time_b, '30m', 
                             LAG(t.energy_exported_phase_c_active_counter_agg) OVER ordered_meter, 
                             LEAD(t.energy_exported_phase_c_active_counter_agg) OVER ordered_meter)
        END AS energy_exported_phase_c_active_delta
        
    FROM flows.mg_meter_readings_30m_intermediate t
    WHERE t.time_b BETWEEN start_time AND end_time
		AND (device_ids IS NULL OR t.device_id = ANY(device_ids))
    WINDOW ordered_meter AS (PARTITION BY t.device_id ORDER BY t.time_b);
END;
$$ LANGUAGE plpgsql;



-- CREATE MATERIALIZED VIEW marcus.bess_readings_5m
-- WITH (timescaledb.continuous) AS
-- select
--     device_id,
--     time_bucket('5m', time) as time_b,
--     avg(soe) as soe_avg,
--     avg(target_power) as target_power_avg
-- FROM marcus.bess_readings
-- GROUP BY device_id, time_b;


-- CREATE MATERIALIZED VIEW marcus.bess_readings_30m
-- WITH (timescaledb.continuous) AS
-- select
--     device_id,
--     time_bucket('30m', time) as time_b,
--     avg(soe) as soe_avg,
--     avg(target_power) as target_power_avg
-- FROM marcus.bess_readings
-- GROUP BY device_id, time_b;

COMMIT;
