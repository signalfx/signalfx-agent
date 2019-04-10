#!/bin/bash

set -euo pipefail

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE DATABASE dvdrental;
	CREATE EXTENSION pg_stat_statements;
EOSQL

psql --username "$POSTGRES_USER" --dbname "dvdrental" < /opt/restore.sql
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "dvdrental" -c "CREATE EXTENSION pg_stat_statements;"

