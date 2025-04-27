#!/bin/bash
source local.env

export MIGRATION_DSN="host=db port=5432 dbname=$DB_NAME user=$DB_USER password=$DB_PASS sslmode=disable"

sleep 2 && goose -dir "${MIGRATION_DIR}" tinvest "${MIGRATION_DSN}" up -v