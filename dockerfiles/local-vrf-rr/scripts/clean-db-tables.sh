#!/bin/bash
psql $DATABASE_URL -c "
DO $$ 
DECLARE 
   r RECORD;
BEGIN
   FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = current_schema()) LOOP
      EXECUTE 'DELETE FROM ' || quote_ident(r.tablename) || ' CASCADE';
   END LOOP;
END $$;"