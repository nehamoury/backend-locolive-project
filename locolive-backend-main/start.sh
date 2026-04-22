#!/bin/sh



set -e



echo "Ensuring volume permissions..."

mkdir -p /app/uploads

chmod 777 /app/uploads



echo "Running database migrations..."

# Attempt to run migrations. If it fails due to a lock, it will log and retry once with a force flag.

if ! /usr/bin/migrate -path /app/db/migrations -database "$DB_SOURCE" -verbose up; then

    echo "Migration failed, checking for locks..."

    # The error message for a migration lock usually contains "dirty" or "locked"

    # We attempt to force it to the current version if a lock is suspected.

    # Note: In production, this should be used with caution.

    VERSION=$(/usr/bin/migrate -path /app/db/migrations -database "$DB_SOURCE" version 2>&1 | grep -oE '[0-9]+' | head -1 || echo "0")

    if [ "$VERSION" != "0" ]; then

        echo "Detected potential migration lock at version $VERSION. Attempting to force-unlock..."

        /usr/bin/migrate -path /app/db/migrations -database "$DB_SOURCE" force "$VERSION"

        echo "Retrying migration up..."

        /usr/bin/migrate -path /app/db/migrations -database "$DB_SOURCE" -verbose up

    else

        echo "Failed to detect migration version, continuing may result in inconsistent state."

        exit 1

    fi

fi



echo "Starting the application..."

exec "$@"