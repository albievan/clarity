#!/bin/bash
# =============================================================================
#  Clarity — SQL Server Custom Entrypoint
#  /clarity/entrypoint.sh
#
#  Sequence:
#    1. Start SQL Server in the background using the official entrypoint
#    2. Poll until SQL Server accepts TCP connections (up to 90 seconds)
#    3. Create login, database, and application user if they don't exist
#    4. Run schema DDL (idempotent — guarded by IF NOT EXISTS checks)
#    5. Trap SIGTERM/SIGINT for graceful shutdown
#    6. Wait for SQL Server PID to exit (keeps container alive)
#
#  Environment variables (all required — set via k8s Secret):
#    MSSQL_SA_PASSWORD     — SQL Server SA (sysadmin) password
#    CLARITY_DB_NAME       — Database name to create (default: clarity)
#    CLARITY_DB_USER       — Application login to create
#    CLARITY_DB_PASSWORD   — Password for the application login
# =============================================================================

set -euo pipefail

SQLCMD="/opt/mssql-tools18/bin/sqlcmd"
INIT_DIR="/clarity/init"
DB_NAME="${CLARITY_DB_NAME:-clarity}"
DB_USER="${CLARITY_DB_USER:-clarity_app}"
DB_PASSWORD="${CLARITY_DB_PASSWORD}"
SA_PASSWORD="${MSSQL_SA_PASSWORD}"
PORT="${MSSQL_TCP_PORT:-1433}"
MAX_WAIT=90
SCHEMA_MARKER="/var/opt/mssql/.clarity_schema_applied"

log() { echo "[$(date -u '+%Y-%m-%dT%H:%M:%SZ')] [clarity-init] $*"; }
err() { log "ERROR: $*" >&2; }

# ---------------------------------------------------------------------------
# Validate required environment variables
# ---------------------------------------------------------------------------
if [[ -z "${SA_PASSWORD:-}" ]]; then
    err "MSSQL_SA_PASSWORD is not set. Cannot start."
    exit 1
fi
if [[ -z "${DB_PASSWORD:-}" ]]; then
    err "CLARITY_DB_PASSWORD is not set. Cannot create application user."
    exit 1
fi

# ---------------------------------------------------------------------------
# Start SQL Server using the official entrypoint in the background
# Pass all arguments through (allows k8s to override CMD)
# ---------------------------------------------------------------------------
log "Starting SQL Server (MSSQL_PID=${MSSQL_PID:-Developer})..."
/opt/mssql/bin/sqlservr &
SQLSERVER_PID=$!

# ---------------------------------------------------------------------------
# Graceful shutdown handler — forward SIGTERM to SQL Server
# ---------------------------------------------------------------------------
shutdown() {
    log "Received shutdown signal. Stopping SQL Server (PID ${SQLSERVER_PID})..."
    kill -SIGTERM "${SQLSERVER_PID}" 2>/dev/null || true
    wait "${SQLSERVER_PID}" 2>/dev/null || true
    log "SQL Server stopped cleanly."
    exit 0
}
trap shutdown SIGTERM SIGINT

# ---------------------------------------------------------------------------
# Wait for SQL Server to become available
# ---------------------------------------------------------------------------
log "Waiting for SQL Server to accept connections on port ${PORT}..."
ELAPSED=0
until ${SQLCMD} \
        -S "localhost,${PORT}" \
        -U sa \
        -P "${SA_PASSWORD}" \
        -Q "SELECT 1" \
        -No \
        -C \
        2>/dev/null; do
    if [[ ${ELAPSED} -ge ${MAX_WAIT} ]]; then
        err "SQL Server did not become ready after ${MAX_WAIT}s. Aborting init."
        exit 1
    fi
    sleep 2
    ELAPSED=$((ELAPSED + 2))
done
log "SQL Server is ready (waited ${ELAPSED}s)."

# ---------------------------------------------------------------------------
# Schema initialisation — only runs on first boot
# A marker file in the persistent volume prevents re-execution on restart
# ---------------------------------------------------------------------------
if [[ -f "${SCHEMA_MARKER}" ]]; then
    log "Schema marker found at ${SCHEMA_MARKER} — skipping initialisation (data persists)."
else
    log "First boot detected — running schema initialisation..."

    # Create the database
    log "Creating database [${DB_NAME}]..."
    ${SQLCMD} -S "localhost,${PORT}" -U sa -P "${SA_PASSWORD}" -No -C -Q "
IF NOT EXISTS (SELECT 1 FROM sys.databases WHERE name = N'${DB_NAME}')
BEGIN
    CREATE DATABASE [${DB_NAME}]
        COLLATE SQL_Latin1_General_CP1_CI_AS;
    PRINT 'Database ${DB_NAME} created.';
END
ELSE
    PRINT 'Database ${DB_NAME} already exists.';
"

    # Create the application login and user
    log "Creating application login [${DB_USER}]..."
    ${SQLCMD} -S "localhost,${PORT}" -U sa -P "${SA_PASSWORD}" -No -C -Q "
USE [master];
IF NOT EXISTS (SELECT 1 FROM sys.server_principals WHERE name = N'${DB_USER}')
BEGIN
    CREATE LOGIN [${DB_USER}] WITH PASSWORD = N'${DB_PASSWORD}',
        CHECK_EXPIRATION = OFF,
        CHECK_POLICY     = ON;
    PRINT 'Login ${DB_USER} created.';
END

USE [${DB_NAME}];
IF NOT EXISTS (SELECT 1 FROM sys.database_principals WHERE name = N'${DB_USER}')
BEGIN
    CREATE USER [${DB_USER}] FOR LOGIN [${DB_USER}];
    -- Grant minimum required permissions to the application user
    GRANT CONNECT      TO [${DB_USER}];
    GRANT SELECT       TO [${DB_USER}];
    GRANT INSERT       TO [${DB_USER}];
    GRANT UPDATE       TO [${DB_USER}];
    GRANT DELETE       TO [${DB_USER}];
    GRANT EXECUTE      TO [${DB_USER}];
    -- Allow user to create tables (needed for first-run schema apply)
    ALTER ROLE db_ddladmin ADD MEMBER [${DB_USER}];
    PRINT 'User ${DB_USER} created and permissions granted.';
END
"

    # Run schema DDL files in lexicographic order
    for SQL_FILE in $(ls "${INIT_DIR}"/*.sql 2>/dev/null | sort); do
        log "Applying: $(basename ${SQL_FILE})..."
        ${SQLCMD} \
            -S "localhost,${PORT}" \
            -U sa \
            -P "${SA_PASSWORD}" \
            -d "${DB_NAME}" \
            -i "${SQL_FILE}" \
            -No \
            -C \
            2>&1 | grep -v "^$" || {
            err "Failed applying $(basename ${SQL_FILE}). Check SQL Server logs."
            exit 1
        }
        log "Applied: $(basename ${SQL_FILE})"
    done

    # Revoke DDL admin after schema is created — principle of least privilege
    ${SQLCMD} -S "localhost,${PORT}" -U sa -P "${SA_PASSWORD}" -No -C -Q "
USE [${DB_NAME}];
ALTER ROLE db_ddladmin DROP MEMBER [${DB_USER}];
PRINT 'DDL admin role revoked from ${DB_USER}.';
"

    # Write marker file to persistent volume
    touch "${SCHEMA_MARKER}"
    log "Schema initialisation complete. Marker written to ${SCHEMA_MARKER}."
fi

# ---------------------------------------------------------------------------
# SQL Server is running — wait until it exits
# ---------------------------------------------------------------------------
log "Clarity SQL Server ready. Waiting on PID ${SQLSERVER_PID}..."
wait "${SQLSERVER_PID}"
EXIT_CODE=$?
log "SQL Server exited with code ${EXIT_CODE}."
exit "${EXIT_CODE}"
