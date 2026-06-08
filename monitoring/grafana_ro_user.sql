-- Utilisateur PostgreSQL en LECTURE SEULE pour la datasource Grafana.
-- À exécuter sur la base control_center :
--   psql "$POSTGRES_DSN" -f monitoring/grafana_ro_user.sql
-- (Adapter le mot de passe, puis le reporter dans monitoring/.env CPM_PG_PASSWORD.)

DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'grafana_ro') THEN
    CREATE ROLE grafana_ro LOGIN PASSWORD 'change-me-too';
  END IF;
END$$;

GRANT CONNECT ON DATABASE control_center TO grafana_ro;
GRANT USAGE ON SCHEMA public TO grafana_ro;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO grafana_ro;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO grafana_ro;
