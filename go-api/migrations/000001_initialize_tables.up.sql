CREATE TABLE IF NOT EXISTS "adapters" (
    adapter_hash TEXT NOT NULL,
    adapter_id BIGSERIAL NOT NULL,
    decimals INTEGER NOT NULL,
    name TEXT NOT NULL,
    CONSTRAINT "adapters_adapter_hash_key" UNIQUE ("adapter_hash"),
    CONSTRAINT "adapters_pkey" PRIMARY KEY ("adapter_id")
);

CREATE TABLE IF NOT EXISTS "chains" (
    chain_id BIGSERIAL NOT NULL,
    name TEXT NOT NULL,
    CONSTRAINT "chains_name_key" UNIQUE ("name"),
    CONSTRAINT "chains_pkey" PRIMARY KEY ("chain_id")
);

CREATE TABLE IF NOT EXISTS "services" (
    name TEXT NOT NULL,
    service_id BIGSERIAL NOT NULL,
    CONSTRAINT "services_name_key" UNIQUE ("name"),
    CONSTRAINT "services_pkey" PRIMARY KEY ("service_id")
);

CREATE TABLE IF NOT EXISTS "aggregators" (
    absolute_threshold DOUBLE PRECISION NOT NULL,
    active BOOLEAN NOT NULL DEFAULT false,
    adapter_id BIGINT NOT NULL,
    address TEXT NOT NULL,
    aggregator_hash TEXT NOT NULL,
    aggregator_id BIGSERIAL NOT NULL,
    chain_id BIGINT NOT NULL,
    fetcher_type INTEGER NOT NULL,
    heartbeat INTEGER NOT NULL,
    name TEXT NOT NULL,
    threshold DOUBLE PRECISION NOT NULL,
    CONSTRAINT "aggregators_address_key" UNIQUE ("address"),
    CONSTRAINT "aggregators_adapter_id_fkey" FOREIGN KEY ("adapter_id") REFERENCES "public"."adapters" ("adapter_id"),
    CONSTRAINT "aggregators_chain_id_fkey" FOREIGN KEY ("chain_id") REFERENCES "public"."chains" ("chain_id"),
    CONSTRAINT "aggregators_pkey" PRIMARY KEY ("aggregator_id")
);

CREATE TABLE IF NOT EXISTS "aggregates" (
    aggregate_id BIGSERIAL NOT NULL,
    aggregator_id BIGINT NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    value BIGINT NOT NULL,
    CONSTRAINT "aggregates_aggregator_id_fkey" FOREIGN KEY ("aggregator_id") REFERENCES "public"."aggregators" ("aggregator_id") ON DELETE CASCADE,
    CONSTRAINT "aggregates_pkey" PRIMARY KEY ("aggregate_id")
);

CREATE TABLE IF NOT EXISTS "feeds" (
    adapter_id BIGINT NOT NULL,
    definition JSONB NOT NULL,
    feed_id BIGSERIAL NOT NULL,
    name TEXT NOT NULL,
    CONSTRAINT "feeds_adapter_id_fkey" FOREIGN KEY ("adapter_id") REFERENCES "public"."adapters" ("adapter_id") ON DELETE CASCADE,
    CONSTRAINT "feeds_pkey" PRIMARY KEY ("feed_id")
);

CREATE TABLE IF NOT EXISTS "data" (
    aggregator_id BIGINT NOT NULL,
    data_id BIGSERIAL NOT NULL,
    feed_id BIGINT NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    value BIGINT NOT NULL,
    CONSTRAINT "data_aggregator_id_fkey" FOREIGN KEY ("aggregator_id") REFERENCES "public"."aggregators" ("aggregator_id") ON DELETE CASCADE,
    CONSTRAINT "data_pkey" PRIMARY KEY ("data_id"),
    CONSTRAINT "data_feed_id_fkey" FOREIGN KEY ("feed_id") REFERENCES "public"."feeds" ("feed_id")
);

CREATE TABLE IF NOT EXISTS "error" (
    code TEXT NOT NULL,
    error_id BIGSERIAL NOT NULL,
    name TEXT NOT NULL,
    request_id TEXT NOT NULL,
    stack TEXT NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT "error_pkey" PRIMARY KEY ("error_id")
);

CREATE TABLE IF NOT EXISTS "listeners" (
    address CHARACTER VARYING(42) NOT NULL,
    chain_id BIGINT NOT NULL,
    event_name CHARACTER VARYING(255) NOT NULL,
    listener_id BIGSERIAL NOT NULL,
    service_id BIGINT NOT NULL,
    CONSTRAINT "listeners_chain_id_fkey" FOREIGN KEY ("chain_id") REFERENCES "public"."chains" ("chain_id"),
    CONSTRAINT "listeners_service_id_fkey" FOREIGN KEY ("service_id") REFERENCES "public"."services" ("service_id"),
    CONSTRAINT "listeners_pkey" PRIMARY KEY ("listener_id")
);

CREATE TABLE IF NOT EXISTS "proxies" (
    host TEXT NOT NULL,
    id BIGSERIAL NOT NULL,
    location TEXT,
    port INTEGER NOT NULL,
    protocol TEXT NOT NULL,
    CONSTRAINT "proxies_protocol_host_port_key" UNIQUE ("protocol", "host", "port"),
    CONSTRAINT "proxies_pkey" PRIMARY KEY ("id")
);

CREATE TABLE IF NOT EXISTS "reporters" (
    address CHARACTER VARYING(42) NOT NULL,
    chain_id BIGINT NOT NULL,
    "oracleAddress" CHARACTER VARYING(42) NOT NULL,
    "privateKey" CHARACTER VARYING(164) NOT NULL,
    reporter_id BIGSERIAL NOT NULL,
    service_id BIGINT NOT NULL,
    CONSTRAINT "reporters_chain_id_fkey" FOREIGN KEY ("chain_id") REFERENCES "public"."chains" ("chain_id"),
    CONSTRAINT "reporters_service_id_fkey" FOREIGN KEY ("service_id") REFERENCES "public"."services" ("service_id"),
    CONSTRAINT "reporters_pkey" PRIMARY KEY ("reporter_id")
);

CREATE TABLE IF NOT EXISTS "vrf_keys" (
    chain_id BIGINT NOT NULL,
    key_hash CHARACTER VARYING(66) NOT NULL,
    pk CHARACTER VARYING(130) NOT NULL,
    pk_x CHARACTER VARYING(78) NOT NULL,
    pk_y CHARACTER VARYING(78) NOT NULL,
    sk CHARACTER VARYING(64) NOT NULL,
    vrf_key_id BIGSERIAL NOT NULL,
    CONSTRAINT "vrf_keys_chain_id_fkey" FOREIGN KEY ("chain_id") REFERENCES "public"."chains" ("chain_id"),
    CONSTRAINT "vrf_keys_pkey" PRIMARY KEY ("vrf_key_id")
);