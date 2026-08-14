CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
CREATE ROLE pgsentinel LOGIN PASSWORD 'pgsentinel-demo-only';
GRANT pg_monitor TO pgsentinel;
CREATE TABLE IF NOT EXISTS public.demo_events(id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY, tenant_id integer NOT NULL, payload jsonb NOT NULL DEFAULT '{}', created_at timestamptz NOT NULL DEFAULT now());
INSERT INTO public.demo_events(tenant_id,payload) SELECT n % 20, jsonb_build_object('sequence',n) FROM generate_series(1,10000) n;
CREATE INDEX demo_events_tenant_idx ON public.demo_events(tenant_id);
ANALYZE public.demo_events;
