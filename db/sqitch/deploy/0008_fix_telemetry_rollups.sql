-- Deploy flux:0008_fix_telemetry_rollups to pg

BEGIN;

-- Add policies for the intermediate rollups which use continous aggregates
SELECT add_continuous_aggregate_policy('flux.mg_meter_readings_30m_intermediate',
    start_offset => null,
    end_offset => null,
    schedule_interval => INTERVAL '5 minutes'
);
SELECT add_continuous_aggregate_policy('flux.mg_meter_readings_5m_intermediate',
    start_offset => null,
    end_offset => null,
    schedule_interval => INTERVAL '5 minutes'
);
SELECT add_continuous_aggregate_policy('flux.mg_bess_readings_5m',
    start_offset => null,
    end_offset => null,
    schedule_interval => INTERVAL '5 minutes'
);
SELECT add_continuous_aggregate_policy('flux.mg_bess_readings_30m',
    start_offset => null,
    end_offset => null,
    schedule_interval => INTERVAL '30 minutes'
);

-- Fix the functiions themselves, which were referencing Flows and had mismatched columns
DROP FUNCTION flux.get_meter_readings_5m(timestamp with time zone,timestamp with time zone,uuid[]);
DROP FUNCTION flux.get_meter_readings_30m(timestamp with time zone,timestamp with time zone,uuid[]);

CREATE OR REPLACE FUNCTION flux.get_meter_readings_5m(
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
    energy_exported_phase_c_active_delta DOUBLE PRECISION,
    energy_imported_reactive_delta DOUBLE PRECISION,
    energy_exported_reactive_delta DOUBLE PRECISION,
    energy_imported_apparent_delta DOUBLE PRECISION,
    energy_exported_apparent_delta DOUBLE PRECISION
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
        END AS energy_exported_phase_c_active_delta,
        
        CASE WHEN t.energy_imported_reactive_counter_agg IS NULL THEN
          NULL 
        ELSE
          interpolated_delta(t.energy_imported_reactive_counter_agg, t.time_b, '5m', 
                             LAG(t.energy_imported_reactive_counter_agg) OVER ordered_meter, 
                             LEAD(t.energy_imported_reactive_counter_agg) OVER ordered_meter)
        END AS energy_imported_reactive_delta,
        
        CASE WHEN t.energy_exported_reactive_counter_agg IS NULL THEN
          NULL 
        ELSE
          interpolated_delta(t.energy_exported_reactive_counter_agg, t.time_b, '5m', 
                             LAG(t.energy_exported_reactive_counter_agg) OVER ordered_meter, 
                             LEAD(t.energy_exported_reactive_counter_agg) OVER ordered_meter)
        END AS energy_exported_reactive_delta,
        
        CASE WHEN t.energy_imported_apparent_counter_agg IS NULL THEN
          NULL 
        ELSE
          interpolated_delta(t.energy_imported_apparent_counter_agg, t.time_b, '5m', 
                             LAG(t.energy_imported_apparent_counter_agg) OVER ordered_meter, 
                             LEAD(t.energy_imported_apparent_counter_agg) OVER ordered_meter)
        END AS energy_imported_apparent_delta,
        
        CASE WHEN t.energy_exported_apparent_counter_agg IS NULL THEN
          NULL 
        ELSE
          interpolated_delta(t.energy_exported_apparent_counter_agg, t.time_b, '5m', 
                             LAG(t.energy_exported_apparent_counter_agg) OVER ordered_meter, 
                             LEAD(t.energy_exported_apparent_counter_agg) OVER ordered_meter)
        END AS energy_exported_apparent_delta
        
    FROM flux.mg_meter_readings_5m_intermediate t
    WHERE t.time_b BETWEEN start_time AND end_time
		AND (device_ids IS NULL OR t.device_id = ANY(device_ids))
    WINDOW ordered_meter AS (PARTITION BY t.device_id ORDER BY t.time_b);
END;
$$ LANGUAGE plpgsql;


-- Do the same with the 30m aggregation
CREATE OR REPLACE FUNCTION flux.get_meter_readings_30m(
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
    energy_exported_phase_c_active_delta DOUBLE PRECISION,
    energy_imported_reactive_delta DOUBLE PRECISION,
    energy_exported_reactive_delta DOUBLE PRECISION,
    energy_imported_apparent_delta DOUBLE PRECISION,
    energy_exported_apparent_delta DOUBLE PRECISION
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
        END AS energy_exported_phase_c_active_delta,
        
        CASE WHEN t.energy_imported_reactive_counter_agg IS NULL THEN
          NULL 
        ELSE
          interpolated_delta(t.energy_imported_reactive_counter_agg, t.time_b, '30m', 
                             LAG(t.energy_imported_reactive_counter_agg) OVER ordered_meter, 
                             LEAD(t.energy_imported_reactive_counter_agg) OVER ordered_meter)
        END AS energy_imported_reactive_delta,
        
        CASE WHEN t.energy_exported_reactive_counter_agg IS NULL THEN
          NULL 
        ELSE
          interpolated_delta(t.energy_exported_reactive_counter_agg, t.time_b, '30m', 
                             LAG(t.energy_exported_reactive_counter_agg) OVER ordered_meter, 
                             LEAD(t.energy_exported_reactive_counter_agg) OVER ordered_meter)
        END AS energy_exported_reactive_delta,
        
        CASE WHEN t.energy_imported_apparent_counter_agg IS NULL THEN
          NULL 
        ELSE
          interpolated_delta(t.energy_imported_apparent_counter_agg, t.time_b, '30m', 
                             LAG(t.energy_imported_apparent_counter_agg) OVER ordered_meter, 
                             LEAD(t.energy_imported_apparent_counter_agg) OVER ordered_meter)
        END AS energy_imported_apparent_delta,
        
        CASE WHEN t.energy_exported_apparent_counter_agg IS NULL THEN
          NULL 
        ELSE
          interpolated_delta(t.energy_exported_apparent_counter_agg, t.time_b, '30m', 
                             LAG(t.energy_exported_apparent_counter_agg) OVER ordered_meter, 
                             LEAD(t.energy_exported_apparent_counter_agg) OVER ordered_meter)
        END AS energy_exported_apparent_delta
        
    FROM flux.mg_meter_readings_30m_intermediate t
    WHERE t.time_b BETWEEN start_time AND end_time
		AND (device_ids IS NULL OR t.device_id = ANY(device_ids))
    WINDOW ordered_meter AS (PARTITION BY t.device_id ORDER BY t.time_b);
END;
$$ LANGUAGE plpgsql;

COMMIT;
