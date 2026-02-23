# =============================================================================
#  Clarity — MariaDB Database Image
#  Base: mariadb:11.4-lts (current Long-Term Support release)
#  Target runtime: k3s on Rocky Linux 10 (x86_64 / aarch64)
#
#  Build:
#    docker build -t clarity-mariadb:1.0.0 .
#
#  Run locally (dev):
#    docker run -d \
#      --name clarity-mariadb \
#      -e MARIADB_ROOT_PASSWORD=ChangeMe_R00t! \
#      -e MARIADB_DATABASE=clarity \
#      -e MARIADB_USER=clarity_app \
#      -e MARIADB_PASSWORD=ChangeMe_App! \
#      -p 3306:3306 \
#      clarity-mariadb:1.0.0
# =============================================================================

FROM mariadb:11.4

# ---------------------------------------------------------------------------
# Labels
# ---------------------------------------------------------------------------
LABEL org.opencontainers.image.title="Clarity MariaDB" \
      org.opencontainers.image.description="Budget Management Platform — MariaDB 11.4 LTS" \
      org.opencontainers.image.version="1.0.0" \
      org.opencontainers.image.vendor="Clarity" \
      maintainer="platform-engineering@clarity.internal"

# ---------------------------------------------------------------------------
# Custom MariaDB configuration
# Mounted at /etc/mysql/conf.d/ — loaded after base my.cnf
# ---------------------------------------------------------------------------
COPY conf/clarity.cnf /etc/mysql/conf.d/clarity.cnf

# ---------------------------------------------------------------------------
# Schema initialisation
# MariaDB will automatically execute .sql files in /docker-entrypoint-initdb.d/
# in lexicographic order when the data directory is empty (first boot only).
#
# File naming convention:
#   00-schema.sql  — DDL: CREATE TABLE, indexes, constraints
#   01-seed.sql    — DML: reference data inserts (currencies, roles, etc.)
#
# On subsequent starts, the init directory is IGNORED — data persists in PVC.
# ---------------------------------------------------------------------------
COPY init/ /docker-entrypoint-initdb.d/

# ---------------------------------------------------------------------------
# Permissions
# All files owned by the mysql user (uid 999) that runs inside the container.
# ---------------------------------------------------------------------------
RUN chown -R 999:999 /docker-entrypoint-initdb.d/ && \
    chmod 640 /docker-entrypoint-initdb.d/*.sql && \
    chmod 644 /etc/mysql/conf.d/clarity.cnf

# ---------------------------------------------------------------------------
# Health check
# (k8s readiness/liveness probes are defined in the StatefulSet)
# ---------------------------------------------------------------------------
HEALTHCHECK \
    --interval=10s \
    --timeout=5s \
    --start-period=30s \
    --retries=5 \
    CMD mariadb-admin ping -h localhost -u root --password="${MARIADB_ROOT_PASSWORD}" || exit 1

# ---------------------------------------------------------------------------
# Runtime defaults — overridden by k8s Secret environment variables
# ---------------------------------------------------------------------------
ENV MARIADB_DATABASE=clarity \
    MARIADB_AUTO_UPGRADE=1 \
    MARIADB_DISABLE_UPGRADE_BACKUP=1

EXPOSE 3306

# Base image USER: mysql (uid 999)
# Base image ENTRYPOINT: mariadbd
# Reads: MARIADB_ROOT_PASSWORD, MARIADB_USER, MARIADB_PASSWORD, MARIADB_DATABASE
