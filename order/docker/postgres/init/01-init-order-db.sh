#!/bin/bash
set -e

# This script is automatically executed by the PostgreSQL Docker image on first startup
# when the data directory is empty. It runs because this file is mounted to
# /docker-entrypoint-initdb.d/ in docker-compose.yml
#
# Creates the orders service database and user with appropriate permissions

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE USER $ORDER_DB_USER WITH PASSWORD '$ORDER_DB_PASSWORD';
    CREATE DATABASE $ORDER_DB_NAME OWNER $ORDER_DB_USER;
    GRANT ALL PRIVILEGES ON DATABASE $ORDER_DB_NAME TO $ORDER_DB_USER;
EOSQL
