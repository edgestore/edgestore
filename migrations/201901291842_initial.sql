CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS records
(
  id uuid DEFAULT uuid_generate_v4() NOT NULL CONSTRAINT records_id_pkey PRIMARY KEY,
  aggregate_id varchar(255) NOT null,
  tenant_id VARCHAR(255) NOT NULL,
  version INTEGER DEFAULT 0 NOT NULL,
  data BYTEA,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT now() NOT NULL,
  CONSTRAINT records_aggregate_id_tenant_id_version UNIQUE (aggregate_id, tenant_id, version)
);

CREATE INDEX IF NOT EXISTS records_aggregate_id_index ON records (aggregate_id);
CREATE INDEX IF NOT EXISTS records_aggregate_id_tenant_id_index ON records (aggregate_id, tenant_id);