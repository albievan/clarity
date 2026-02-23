# =============================================================================
#  Clarity — SQL Server 2022 Database Image
#  Base: mcr.microsoft.com/mssql/server:2022-latest (x86_64 only)
#  Target runtime: k3s on Rocky Linux 10 (x86_64)
#
#  NOTE: SQL Server for Linux is x86_64 only. aarch64 (ARM) is NOT supported.
#        Rocky Linux 10 on standard x86_64 hardware is fully supported.
#
#  Build:
#    docker build -t clarity-sqlserver:1.0.0 .
#
#  Run locally (dev):
#    docker run -d \
#      --name clarity-sqlserver \
#      -e ACCEPT_EULA=Y \
#      -e MSSQL_SA_PASSWORD='ChangeMe_SA_1!' \
#      -e MSSQL_PID=Developer \
#      -e CLARITY_DB_NAME=clarity \
#      -e CLARITY_DB_USER=clarity_app \
#      -e CLARITY_DB_PASSWORD='ChangeMe_App_1!' \
#      -p 1433:1433 \
#      clarity-sqlserver:1.0.0
#
#  Licence note:
#    MSSQL_PID=Developer — free for non-production use
#    MSSQL_PID=Standard  — requires licence
#    MSSQL_PID=Enterprise — requires licence
#    MSSQL_PID=Express   — free, limited to 10 GB data / 1 GB RAM
# =============================================================================

FROM mcr.microsoft.com/mssql/server:2022-latest

# ---------------------------------------------------------------------------
# Labels
# ---------------------------------------------------------------------------
LABEL org.opencontainers.image.title="Clarity SQL Server" \
      org.opencontainers.image.description="Budget Management Platform — SQL Server 2022" \
      org.opencontainers.image.version="1.0.0" \
      org.opencontainers.image.vendor="Clarity" \
      maintainer="platform-engineering@clarity.internal"

# ---------------------------------------------------------------------------
# The SQL Server base image runs as root by default but drops to mssql
# (uid 10001) for the actual database process.
# We switch to root temporarily to copy files, then switch back.
# ---------------------------------------------------------------------------
USER root

# ---------------------------------------------------------------------------
# Install sqlcmd (already present in 2022 image) and any extras
# The 2022 image uses Ubuntu 22.04 as base OS
# ---------------------------------------------------------------------------
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        curl \
        mssql-tools18 \
        unixodbc-dev && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# ---------------------------------------------------------------------------
# Schema initialisation scripts
# ---------------------------------------------------------------------------
RUN mkdir -p /clarity/init && chown -R 10001:0 /clarity
COPY --chown=10001:0 init/ /clarity/init/

# ---------------------------------------------------------------------------
# Custom entrypoint script
# Wraps the official SQL Server entrypoint to:
#   1. Start SQL Server in the background
#   2. Wait until it's accepting connections
#   3. Run schema initialisation SQL (once only — guarded by DB existence check)
#   4. Bring SQL Server to the foreground
# ---------------------------------------------------------------------------
COPY --chown=10001:0 init/entrypoint.sh /clarity/entrypoint.sh
RUN chmod +x /clarity/entrypoint.sh

# ---------------------------------------------------------------------------
# Switch to the mssql user for runtime
# uid=10001 matches SQL Server's internal process user
# ---------------------------------------------------------------------------
USER 10001

# ---------------------------------------------------------------------------
# Environment defaults — all overridden by k8s Secret at runtime
# ---------------------------------------------------------------------------
# ACCEPT_EULA must be Y — required by Microsoft
ENV ACCEPT_EULA=Y \
    MSSQL_PID=Developer \
    MSSQL_TCP_PORT=1433 \
    MSSQL_COLLATION=SQL_Latin1_General_CP1_CI_AS \
    MSSQL_LCID=1033 \
    MSSQL_MEMORY_LIMIT_MB=2048 \
    CLARITY_DB_NAME=clarity

# ---------------------------------------------------------------------------
# Volumes
# SQL Server stores data in /var/opt/mssql — mapped to PVC in k8s
# ---------------------------------------------------------------------------
VOLUME ["/var/opt/mssql"]

EXPOSE 1433

# ---------------------------------------------------------------------------
# Health check
# (k8s probes are also defined in the StatefulSet for richer control)
# ---------------------------------------------------------------------------
HEALTHCHECK \
    --interval=10s \
    --timeout=5s \
    --start-period=60s \
    --retries=10 \
    CMD /opt/mssql-tools18/bin/sqlcmd \
        -S localhost,${MSSQL_TCP_PORT} \
        -U sa \
        -P "${MSSQL_SA_PASSWORD}" \
        -Q "SELECT 1" \
        -No \
        -C \
        2>/dev/null || exit 1

ENTRYPOINT ["/clarity/entrypoint.sh"]
