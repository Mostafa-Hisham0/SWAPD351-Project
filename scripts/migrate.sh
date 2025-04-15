#!/bin/bash

# Check if we're running in Docker
if [ -f /.dockerenv ]; then
    # Inside Docker
    echo "Running migrations inside Docker..."
    psql $DATABASE_URL -f /app/migrations/001_schema.sql
else
    # Outside Docker
    echo "Waiting for PostgreSQL to be ready..."
    until docker-compose exec postgres pg_isready -U postgres; do
        echo "PostgreSQL is unavailable - sleeping"
        sleep 1
    done

    echo "Running migrations..."
    docker-compose exec postgres psql -U postgres -d rtcs -f /migrations/001_schema.sql
fi

echo "Migrations completed successfully!" 