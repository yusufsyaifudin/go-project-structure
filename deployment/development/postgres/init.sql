-- wallet_db is created by POSTGRES_DB env var; create grafana_db here
CREATE DATABASE grafana_db;
GRANT ALL PRIVILEGES ON DATABASE grafana_db TO wallet;
