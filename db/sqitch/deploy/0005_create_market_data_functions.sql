-- Deploy flux:create-market-data-functions to pg

BEGIN;


create type flux.market_data_input as (time timestamptz, type integer, value float4);

CREATE FUNCTION flux.insert_market_data_batch(data flux.market_data_input[]) RETURNS void
    LANGUAGE plpgsql
    AS $$
DECLARE
    last_value FLOAT;
    data_record flux.market_data_input;
    epsilon FLOAT := 1e-6;  -- Define a small threshold for "nearly equal"
BEGIN
     -- Loop through each row in the input array
    FOREACH data_record IN ARRAY data LOOP

        -- Get the last inserted value (most recent)
        SELECT value INTO last_value
	    FROM flux.market_data
	    WHERE type = data_record.type and time = data_record.time
	    ORDER BY created_at DESC
	    LIMIT 1;

	    -- If the table is empty or the value has changed, insert the new row
        IF last_value IS NULL OR ABS(last_value - data_record.value) > epsilon THEN
            INSERT INTO flux.market_data (time, type, value)
            VALUES (data_record.time, data_record.type, data_record.value);
        END IF;
    END LOOP;
END;
$$;

ALTER FUNCTION flux.insert_market_data_batch(data flux.market_data_input[]) OWNER TO tsdbadmin;

COMMIT;
