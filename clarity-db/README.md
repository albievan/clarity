# Clarity — Database Containers for k3s on Rocky Linux 10

Two production-ready database images with full Kubernetes manifests for k3s.

---

## Contents

```
clarity-db/
├── mariadb/
│   ├── Dockerfile              MariaDB 11.4 LTS image
│   ├── conf/clarity.cnf        Tuned server configuration
│   └── init/00-schema.sql      Clarity schema (auto-applied on first boot)
├── sqlserver/
│   ├── Dockerfile              SQL Server 2022 image
│   └── init/
│       ├── entrypoint.sh       Custom startup: waits for SQL Server, runs schema
│       └── 00-schema.sql       Clarity schema
├── k8s/
│   ├── 00-namespace.yaml       clarity-db namespace
│   ├── mariadb/
│   │   ├── 01-secret.yaml      Root + app passwords
│   │   ├── 02-configmap.yaml   Non-secret env vars
│   │   ├── 03-pvc.yaml         20Gi local-path PVC
│   │   ├── 04-statefulset.yaml StatefulSet with SELinux-compatible securityContext
│   │   └── 05-service.yaml     ClusterIP + headless services
│   ├── sqlserver/
│   │   ├── 01-secret.yaml      SA + app passwords
│   │   ├── 02-pvc.yaml         30Gi local-path PVC
│   │   ├── 03-statefulset.yaml StatefulSet (x86_64 only, 2–4GB RAM)
│   │   └── 04-service.yaml     ClusterIP + headless services
│   └── kustomization.yaml
└── scripts/
    ├── build.sh                Build + push/import images
    └── deploy.sh               Apply manifests + wait for ready
```

---

## Prerequisites

### Rocky Linux 10 — k3s installation

```bash
# Install k3s
curl -sfL https://get.k3s.io | sh -

# Configure kubectl for your user
mkdir -p ~/.kube
sudo cp /etc/rancher/k3s/k3s.yaml ~/.kube/config
sudo chown $(id -u):$(id -g) ~/.kube/config
export KUBECONFIG=~/.kube/config

# Verify
kubectl get nodes
```

### SELinux (Rocky Linux 10 default: enforcing)

k3s and the local-path provisioner work with SELinux enforcing. The manifests
use `securityContext.fsGroup` and init containers to ensure volume directories
are correctly owned before the database process starts.

If you see `Permission denied` on volume mounts:
```bash
# Check SELinux denials
sudo ausearch -m avc -ts recent | grep k3s

# Label the k3s storage directory if needed
sudo semanage fcontext -a -t container_file_t "/var/lib/rancher/k3s/storage(/.*)?"
sudo restorecon -Rv /var/lib/rancher/k3s/storage
```

---

## Quick Start — Single Node (no registry)

This approach builds images locally and imports them directly into k3s
containerd. No container registry required.

```bash
# 1. Build images
./scripts/build.sh --import-k3s

# 2. Update StatefulSets to use imagePullPolicy: Never
#    (already set to IfNotPresent — k3s will use local image)

# 3. Change passwords in the Secret files (REQUIRED before deploying)
#    See "Changing Passwords" section below.

# 4. Deploy
./scripts/deploy.sh
```

---

## Production Setup — With a Registry

```bash
# 1. Build and push
./scripts/build.sh \
  --registry registry.your-company.com/clarity \
  --tag 1.0.0 \
  --push

# 2. Update image names in the StatefulSets:
#    k8s/mariadb/04-statefulset.yaml → image: registry.your-company.com/clarity/clarity-mariadb:1.0.0
#    k8s/sqlserver/03-statefulset.yaml → image: registry.your-company.com/clarity/clarity-sqlserver:1.0.0

# 3. If registry requires auth, create an imagePullSecret:
kubectl create secret docker-registry clarity-registry \
  --namespace clarity-db \
  --docker-server=registry.your-company.com \
  --docker-username=YOUR_USER \
  --docker-password=YOUR_PASSWORD

# Then add to StatefulSpec.spec.imagePullSecrets:
#   imagePullSecrets:
#     - name: clarity-registry

# 4. Change passwords (see below)

# 5. Deploy
./scripts/deploy.sh
```

### Optional: Local registry for k3s

```bash
# Start a local registry
docker run -d -p 5000:5000 --restart=always --name registry registry:2

# Tell k3s about it — create /etc/rancher/k3s/registries.yaml:
sudo tee /etc/rancher/k3s/registries.yaml << EOF
mirrors:
  "localhost:5000":
    endpoint:
      - "http://localhost:5000"
EOF

# Restart k3s to pick up the registry config
sudo systemctl restart k3s

# Build and push to local registry
./scripts/build.sh --registry localhost:5000/clarity --push
```

---

## Changing Passwords (Required Before Production Deployment)

The Secret YAML files contain placeholder base64-encoded passwords. Replace
them with your own before deploying.

```bash
# Generate a strong password
openssl rand -base64 32

# Base64-encode it for the Secret YAML
echo -n 'YOUR_STRONG_PASSWORD_HERE' | base64

# Update k8s/mariadb/01-secret.yaml and k8s/sqlserver/01-secret.yaml
```

For production, use Sealed Secrets or External Secrets Operator instead of
plaintext Secrets in version control:

```bash
# Install Sealed Secrets controller
kubectl apply -f https://github.com/bitnami-labs/sealed-secrets/releases/latest/download/controller.yaml

# Seal a secret
kubeseal --format yaml < k8s/mariadb/01-secret.yaml > k8s/mariadb/01-sealed-secret.yaml
```

---

## Connection Details

After deployment, both databases are accessible within the cluster:

| Database   | Host                                                     | Port |
|------------|----------------------------------------------------------|------|
| MariaDB    | `clarity-mariadb.clarity-db.svc.cluster.local`          | 3306 |
| SQL Server | `clarity-sqlserver.clarity-db.svc.cluster.local`        | 1433 |

### MariaDB connection string
```
mysql://clarity_app:<password>@clarity-mariadb.clarity-db.svc.cluster.local:3306/clarity?charset=utf8mb4
```

### SQL Server connection string
```
Server=clarity-sqlserver.clarity-db.svc.cluster.local,1433;Database=clarity;User Id=clarity_app;Password=<password>;TrustServerCertificate=True;
```

---

## Useful Commands

```bash
# Watch pods come up
kubectl get pods -n clarity-db -w

# Tail MariaDB logs (schema init happens here on first boot)
kubectl logs -n clarity-db clarity-mariadb-0 -f

# Tail SQL Server logs
kubectl logs -n clarity-db clarity-sqlserver-0 -f

# Connect to MariaDB shell
kubectl exec -it -n clarity-db clarity-mariadb-0 -- \
  mariadb -u clarity_app -p clarity

# Connect to SQL Server via sqlcmd
kubectl exec -it -n clarity-db clarity-sqlserver-0 -- \
  /opt/mssql-tools18/bin/sqlcmd -S localhost -U sa -P "$SA_PASSWORD" -d clarity -No -C

# Check PVC usage
kubectl exec -it -n clarity-db clarity-mariadb-0 -- df -h /var/lib/mysql
kubectl exec -it -n clarity-db clarity-sqlserver-0 -- df -h /var/opt/mssql

# Force re-run schema (MariaDB — only if you know what you're doing)
kubectl exec -it -n clarity-db clarity-mariadb-0 -- bash
# Inside: drop and recreate the database, then delete /var/lib/mysql/.schema_applied (not applicable — MariaDB re-runs on empty datadir only)

# Force re-run schema (SQL Server — delete the marker file)
kubectl exec -it -n clarity-db clarity-sqlserver-0 -- rm /var/opt/mssql/.clarity_schema_applied
kubectl rollout restart statefulset/clarity-sqlserver -n clarity-db
```

---

## Storage

Both databases use k3s's built-in `local-path` storage class by default.

| | MariaDB | SQL Server |
|---|---|---|
| PVC size | 20 Gi | 30 Gi |
| Mount point | `/var/lib/mysql` | `/var/opt/mssql` |
| Storage class | `local-path` | `local-path` |

**For HA/multi-node clusters**, replace `local-path` with Longhorn:
```bash
kubectl apply -f https://raw.githubusercontent.com/longhorn/longhorn/master/deploy/longhorn.yaml
# Then change storageClassName to "longhorn" in the PVC YAML files
```

---

## Resource Requirements

| | Requests | Limits |
|---|---|---|
| **MariaDB** memory | 768 Mi | 1.5 Gi |
| **MariaDB** CPU | 250m | 1000m |
| **SQL Server** memory | 2 Gi | 4 Gi |
| **SQL Server** CPU | 500m | 2000m |

SQL Server **requires** at least 2 GB RAM. Setting the limit below 2 Gi
will cause the SQL Server process to be OOMKilled immediately.

---

## Architecture Notes

- **StatefulSets** are used instead of Deployments to give each pod a stable
  network identity (`clarity-mariadb-0`, `clarity-sqlserver-0`) and ensure
  ordered, graceful startup and shutdown.

- **Schema initialisation** runs automatically on first boot only. For
  MariaDB, the official entrypoint runs `.sql` files from
  `/docker-entrypoint-initdb.d/`. For SQL Server, a custom `entrypoint.sh`
  starts the server, waits for it to accept connections, applies the schema,
  then writes a marker file to the PVC to prevent re-execution on restart.

- **SELinux compatibility**: Init containers fix volume ownership before the
  database process starts (`chown -R <uid>:<gid> /data-path`). This is
  necessary because k3s's local-path provisioner creates directories owned by
  root, and SELinux-enforcing Rocky Linux 10 will deny writes without correct
  ownership.

- **SQL Server platform**: SQL Server for Linux is **x86_64 only**. The
  StatefulSet includes a `nodeSelector: kubernetes.io/arch: amd64` to prevent
  scheduling on ARM nodes in mixed clusters.
