#!/bin/sh
set -e

DB_HOST="${DB_HOST:-db}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-check_status}"
DB_USER="${POSTGRES_USER:-postgres}"
export PGPASSWORD="${POSTGRES_PASSWORD:-}"

# List databases and check whether ours exists.
if ! psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -w \
         -lqt 2>/dev/null | cut -d'|' -f1 | grep -qw "$DB_NAME"; then
    echo "Database '$DB_NAME' not found — creating and seeding..."
    createdb -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -w "$DB_NAME"
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -w -d "$DB_NAME" \
         -f /app/schema.sql
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -w -d "$DB_NAME" -c "
        INSERT INTO websites (url, name) VALUES
            ('https://example.com',    'Example'),
            ('https://httpstat.us/503','Always Down (503)');"
    echo "Done."
fi

export DATABASE_URL="${DATABASE_URL:-postgres://${DB_USER}:${POSTGRES_PASSWORD:-}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable}"
exec /app/check_status
