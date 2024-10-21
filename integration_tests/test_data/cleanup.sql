-- Begin a transaction
BEGIN;

-- Disable triggers to avoid foreign key constraint checks during truncation
SET session_replication_role = replica;

-- Truncate all tables and restart identity sequences
DO
$$
    DECLARE
        rec RECORD;
    BEGIN
        -- Loop through all tables in the 'public' schema
        FOR rec IN
            SELECT tablename
            FROM pg_tables
            WHERE schemaname = 'public'
            LOOP
                -- Execute the TRUNCATE command on each table
                EXECUTE 'TRUNCATE TABLE public.' || quote_ident(rec.tablename) || ' RESTART IDENTITY CASCADE;';
            END LOOP;
    END;
$$;

-- Re-enable triggers
SET session_replication_role = DEFAULT;

-- Commit the transaction
COMMIT;
