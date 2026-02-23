#!/usr/bin/env bash
# =============================================================================
#  Clarity DB — Build & Push Container Images
#
#  Usage:
#    ./scripts/build.sh [OPTIONS]
#
#  Options:
#    --registry  REGISTRY   Image registry prefix (default: localhost:5000/clarity)
#    --tag       TAG        Image tag (default: 1.0.0)
#    --push                 Push images after build (default: false)
#    --mariadb-only         Build only MariaDB image
#    --sqlserver-only       Build only SQL Server image
#    --import-k3s           Import images into k3s containerd after build
#
#  Examples:
#    # Build locally (no push)
#    ./scripts/build.sh
#
#    # Build and push to private registry
#    ./scripts/build.sh --registry registry.clarity.internal/clarity --push
#
#    # Build and import directly into k3s (single-node, no registry needed)
#    ./scripts/build.sh --import-k3s
# =============================================================================

set -euo pipefail

# ---------------------------------------------------------------------------
# Defaults
# ---------------------------------------------------------------------------
REGISTRY="${CLARITY_REGISTRY:-localhost:5000/clarity}"
TAG="${CLARITY_TAG:-1.0.0}"
PUSH=false
IMPORT_K3S=false
BUILD_MARIADB=true
BUILD_SQLSERVER=true
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "${SCRIPT_DIR}")"

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------
while [[ $# -gt 0 ]]; do
    case $1 in
        --registry)      REGISTRY="$2";     shift 2 ;;
        --tag)           TAG="$2";           shift 2 ;;
        --push)          PUSH=true;          shift ;;
        --import-k3s)    IMPORT_K3S=true;    shift ;;
        --mariadb-only)  BUILD_SQLSERVER=false; shift ;;
        --sqlserver-only) BUILD_MARIADB=false; shift ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

MARIADB_IMAGE="${REGISTRY}/clarity-mariadb:${TAG}"
SQLSERVER_IMAGE="${REGISTRY}/clarity-sqlserver:${TAG}"

log() { echo "[$(date '+%H:%M:%S')] $*"; }
hr()  { echo "──────────────────────────────────────────────────────────"; }

hr
log "Clarity DB — Image Build Script"
log "Registry : ${REGISTRY}"
log "Tag      : ${TAG}"
log "Push     : ${PUSH}"
log "Import k3s: ${IMPORT_K3S}"
hr

# ---------------------------------------------------------------------------
# Build MariaDB image
# ---------------------------------------------------------------------------
if [[ "${BUILD_MARIADB}" == "true" ]]; then
    log "Building MariaDB image: ${MARIADB_IMAGE}"
    docker build \
        --file "${ROOT_DIR}/mariadb/Dockerfile" \
        --tag "${MARIADB_IMAGE}" \
        --tag "${REGISTRY}/clarity-mariadb:latest" \
        --label "build.date=$(date -u '+%Y-%m-%dT%H:%M:%SZ')" \
        --label "build.git-commit=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')" \
        "${ROOT_DIR}/mariadb"
    log "MariaDB build complete: ${MARIADB_IMAGE}"
fi

# ---------------------------------------------------------------------------
# Build SQL Server image
# ---------------------------------------------------------------------------
if [[ "${BUILD_SQLSERVER}" == "true" ]]; then
    log "Building SQL Server image: ${SQLSERVER_IMAGE}"
    docker build \
        --file "${ROOT_DIR}/sqlserver/Dockerfile" \
        --tag "${SQLSERVER_IMAGE}" \
        --tag "${REGISTRY}/clarity-sqlserver:latest" \
        --label "build.date=$(date -u '+%Y-%m-%dT%H:%M:%SZ')" \
        --label "build.git-commit=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')" \
        "${ROOT_DIR}/sqlserver"
    log "SQL Server build complete: ${SQLSERVER_IMAGE}"
fi

# ---------------------------------------------------------------------------
# Push to registry
# ---------------------------------------------------------------------------
if [[ "${PUSH}" == "true" ]]; then
    hr
    log "Pushing images to registry..."
    if [[ "${BUILD_MARIADB}" == "true" ]]; then
        docker push "${MARIADB_IMAGE}"
        docker push "${REGISTRY}/clarity-mariadb:latest"
        log "Pushed: ${MARIADB_IMAGE}"
    fi
    if [[ "${BUILD_SQLSERVER}" == "true" ]]; then
        docker push "${SQLSERVER_IMAGE}"
        docker push "${REGISTRY}/clarity-sqlserver:latest"
        log "Pushed: ${SQLSERVER_IMAGE}"
    fi
fi

# ---------------------------------------------------------------------------
# Import directly into k3s containerd (no registry required)
# Useful for single-node k3s dev/test clusters
# ---------------------------------------------------------------------------
if [[ "${IMPORT_K3S}" == "true" ]]; then
    hr
    log "Importing images into k3s containerd..."

    if [[ "${BUILD_MARIADB}" == "true" ]]; then
        MARIADB_TAR="/tmp/clarity-mariadb-${TAG}.tar"
        log "Saving MariaDB image to ${MARIADB_TAR}..."
        docker save "${MARIADB_IMAGE}" -o "${MARIADB_TAR}"
        log "Importing into k3s..."
        k3s ctr images import "${MARIADB_TAR}"
        rm -f "${MARIADB_TAR}"
        log "MariaDB image imported into k3s containerd"
    fi

    if [[ "${BUILD_SQLSERVER}" == "true" ]]; then
        SQLSERVER_TAR="/tmp/clarity-sqlserver-${TAG}.tar"
        log "Saving SQL Server image to ${SQLSERVER_TAR}..."
        docker save "${SQLSERVER_IMAGE}" -o "${SQLSERVER_TAR}"
        log "Importing into k3s..."
        k3s ctr images import "${SQLSERVER_TAR}"
        rm -f "${SQLSERVER_TAR}"
        log "SQL Server image imported into k3s containerd"
    fi
fi

hr
log "Build complete."

if [[ "${PUSH}" == "false" && "${IMPORT_K3S}" == "false" ]]; then
    echo ""
    echo "  Images are available in local Docker daemon."
    echo "  To deploy:"
    echo ""
    echo "  Option A — Push to registry, then deploy:"
    echo "    ./scripts/build.sh --registry YOUR_REGISTRY --push"
    echo "    # Update image names in k8s StatefulSets, then:"
    echo "    ./scripts/deploy.sh"
    echo ""
    echo "  Option B — Import directly into k3s (single node only):"
    echo "    ./scripts/build.sh --import-k3s"
    echo "    # Then update k8s StatefulSets to use imagePullPolicy: Never"
    echo "    ./scripts/deploy.sh"
fi
