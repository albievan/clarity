#!/usr/bin/env bash
# =============================================================================
#  Clarity DB — Deploy to k3s
#
#  Prerequisites:
#    - k3s running on Rocky Linux 10
#    - kubectl configured (k3s writes kubeconfig to /etc/rancher/k3s/k3s.yaml)
#    - Images built and available (via registry or k3s ctr import)
#
#  Usage:
#    ./scripts/deploy.sh [--mariadb-only | --sqlserver-only] [--dry-run]
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "${SCRIPT_DIR}")"
K8S_DIR="${ROOT_DIR}/k8s"

DEPLOY_MARIADB=true
DEPLOY_SQLSERVER=true
DRY_RUN=false
KUBECTL="${KUBECTL:-kubectl}"

while [[ $# -gt 0 ]]; do
    case $1 in
        --mariadb-only)   DEPLOY_SQLSERVER=false; shift ;;
        --sqlserver-only) DEPLOY_MARIADB=false;   shift ;;
        --dry-run)        DRY_RUN=true;           shift ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

log() { echo "[$(date '+%H:%M:%S')] $*"; }
hr()  { echo "──────────────────────────────────────────────────────────"; }

KUBECTL_CMD="${KUBECTL}"
if [[ "${DRY_RUN}" == "true" ]]; then
    KUBECTL_CMD="${KUBECTL} --dry-run=client"
    log "DRY RUN MODE — no changes will be made"
fi

hr
log "Clarity DB — k3s Deployment"
log "MariaDB : ${DEPLOY_MARIADB}"
log "SQL Server: ${DEPLOY_SQLSERVER}"
hr

# ---------------------------------------------------------------------------
# Namespace
# ---------------------------------------------------------------------------
log "Applying namespace..."
${KUBECTL_CMD} apply -f "${K8S_DIR}/00-namespace.yaml"

# ---------------------------------------------------------------------------
# MariaDB
# ---------------------------------------------------------------------------
if [[ "${DEPLOY_MARIADB}" == "true" ]]; then
    hr
    log "Deploying MariaDB..."
    ${KUBECTL_CMD} apply -f "${K8S_DIR}/mariadb/01-secret.yaml"
    ${KUBECTL_CMD} apply -f "${K8S_DIR}/mariadb/02-configmap.yaml"
    ${KUBECTL_CMD} apply -f "${K8S_DIR}/mariadb/03-pvc.yaml"
    ${KUBECTL_CMD} apply -f "${K8S_DIR}/mariadb/04-statefulset.yaml"
    ${KUBECTL_CMD} apply -f "${K8S_DIR}/mariadb/05-service.yaml"
    ${KUBECTL_CMD} apply -f "${K8S_DIR}/mariadb/06-headless-service.yaml"

    if [[ "${DRY_RUN}" == "false" ]]; then
        log "Waiting for MariaDB to become ready (up to 3 minutes)..."
        ${KUBECTL} rollout status statefulset/clarity-mariadb \
            -n clarity-db \
            --timeout=180s && \
        log "MariaDB is ready." || \
        log "WARNING: Rollout timed out. Check: kubectl logs -n clarity-db clarity-mariadb-0"
    fi
fi

# ---------------------------------------------------------------------------
# SQL Server
# ---------------------------------------------------------------------------
if [[ "${DEPLOY_SQLSERVER}" == "true" ]]; then
    hr
    log "Deploying SQL Server..."
    ${KUBECTL_CMD} apply -f "${K8S_DIR}/sqlserver/01-secret.yaml"
    ${KUBECTL_CMD} apply -f "${K8S_DIR}/sqlserver/02-pvc.yaml"
    ${KUBECTL_CMD} apply -f "${K8S_DIR}/sqlserver/03-statefulset.yaml"
    ${KUBECTL_CMD} apply -f "${K8S_DIR}/sqlserver/04-service.yaml"
    ${KUBECTL_CMD} apply -f "${K8S_DIR}/sqlserver/05-nodeport.yaml"

    if [[ "${DRY_RUN}" == "false" ]]; then
        log "Waiting for SQL Server to become ready (up to 10 minutes)..."
        ${KUBECTL} rollout status statefulset/clarity-sqlserver \
            -n clarity-db \
            --timeout=600s && \
        log "SQL Server is ready." || \
        log "WARNING: Rollout timed out. Check: kubectl logs -n clarity-db clarity-sqlserver-0"
    fi
fi

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
if [[ "${DRY_RUN}" == "false" ]]; then
    hr
    log "Deployment complete. Pod status:"
    ${KUBECTL} get pods -n clarity-db -o wide
    echo ""
    log "Services:"
    ${KUBECTL} get svc -n clarity-db
    echo ""
    log "Persistent volumes:"
    ${KUBECTL} get pvc -n clarity-db
    echo ""
    log "Connection endpoints:"
    if [[ "${DEPLOY_MARIADB}" == "true" ]]; then
        echo "  MariaDB  : clarity-mariadb.clarity-db.svc.cluster.local:3306"
    fi
    if [[ "${DEPLOY_SQLSERVER}" == "true" ]]; then
        echo "  SQL Server: clarity-sqlserver.clarity-db.svc.cluster.local:1433"
    fi
    echo ""
    log "To tail logs:"
    [[ "${DEPLOY_MARIADB}" == "true" ]]    && echo "  kubectl logs -n clarity-db clarity-mariadb-0 -f"
    [[ "${DEPLOY_SQLSERVER}" == "true" ]]  && echo "  kubectl logs -n clarity-db clarity-sqlserver-0 -f"
fi
