-- Drop foreign key constraints first
ALTER TABLE IF EXISTS "feeds" DROP CONSTRAINT IF EXISTS "feeds_adapter_id_fkey";
ALTER TABLE IF EXISTS "aggregates" DROP CONSTRAINT IF EXISTS "aggregates_aggregator_id_fkey";
ALTER TABLE IF EXISTS "aggregators" DROP CONSTRAINT IF EXISTS "aggregators_adapter_id_fkey";
ALTER TABLE IF EXISTS "aggregators" DROP CONSTRAINT IF EXISTS "aggregators_chain_id_fkey";
ALTER TABLE IF EXISTS "data" DROP CONSTRAINT IF EXISTS "data_aggregator_id_fkey";
ALTER TABLE IF EXISTS "data" DROP CONSTRAINT IF EXISTS "data_feed_id_fkey";
ALTER TABLE IF EXISTS "listeners" DROP CONSTRAINT IF EXISTS "listeners_chain_id_fkey";
ALTER TABLE IF EXISTS "listeners" DROP CONSTRAINT IF EXISTS "listeners_service_id_fkey";
ALTER TABLE IF EXISTS "reporters" DROP CONSTRAINT IF EXISTS "reporters_chain_id_fkey";
ALTER TABLE IF EXISTS "reporters" DROP CONSTRAINT IF EXISTS "reporters_service_id_fkey";

-- Drop tables in reverse order of creation
DROP TABLE IF EXISTS "vrf_keys";
DROP TABLE IF EXISTS "services";
DROP TABLE IF EXISTS "reporters";
DROP TABLE IF EXISTS "proxies";
DROP TABLE IF EXISTS "listeners";
DROP TABLE IF EXISTS "feeds";
DROP TABLE IF EXISTS "error";
DROP TABLE IF EXISTS "data";
DROP TABLE IF EXISTS "chains";
DROP TABLE IF EXISTS "aggregators";
DROP TABLE IF EXISTS "aggregates";
DROP TABLE IF EXISTS "adapters";
