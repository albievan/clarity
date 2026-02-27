# Clarity API — Go Framework

**Budget Management Platform · Internal REST API**
`github.com/albievan/clarity/clarity-api`

---

## Overview

This repository contains the Go API framework for the **Clarity Budget Management Platform**. It implements a hexagonal (ports and adapters) architecture across 27 domain packages, covering all 141 endpoints defined in the Clarity OpenAPI specification.

The framework is deliberately structured to separate concerns cleanly so that developers can implement SQL queries, business rules, and HTTP handling independently of each other.

---

## Prerequisites

The following tools must be installed on your development machine before building or running the API.

### Required

| Tool | Minimum Version | Purpose | Install |
|------|----------------|---------|---------|
| **Go** | 1.22 | Compiler and toolchain | https://go.dev/dl/ |
| **Git** | 2.x | Source control | https://git-scm.com/ |
| **PostgreSQL** or **MariaDB** | PG 15+ / MariaDB 11.4+ | Primary database | See [Database](#database) |

### Recommended for Development

| Tool | Purpose | Install |
|------|---------|---------|
| **Docker** + **Docker Compose** | Run DB and Redis locally without installing them | https://docs.docker.com/get-docker/ |
| **Redis** | Session caching, rate limiting, idempotency store | https://redis.io/download |
| **golangci-lint** | Static analysis and linting | https://golangci-lint.run/usage/install/ |
| **MinIO** | Local S3-compatible object storage for document uploads | https://min.io/download |
| **curl** or **Postman** | API testing | https://www.postman.com/ |

### Verifying Your Go Installation

```bash
go version
# Expected: go version go1.22.x linux/amd64 (or darwin/arm64 etc.)

go env GOPATH
# Ensure $GOPATH/bin is on your PATH
```

---

## Getting Started

### 1. Clone the Repository

```bash
git clone https://github.com/albievan/clarity/clarity-api.git
cd clarity-api
```

### 2. Install Go Dependencies

```bash
go mod download
```

This will fetch all declared dependencies from `go.mod`. No internet access is required after this step. The following external packages are used:

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/go-chi/chi/v5` | v5.0.14 | HTTP router and middleware |
| `github.com/go-chi/cors` | v1.2.1 | CORS middleware |
| `github.com/golang-jwt/jwt/v5` | v5.2.1 | JWT signing and parsing |
| `github.com/google/uuid` | v1.6.0 | UUID generation |
| `github.com/lib/pq` | v1.10.9 | PostgreSQL driver (also works with CockroachDB) |

### 3. Configure Environment Variables

Copy the example environment file and fill in your values:

```bash
cp .env.example .env
```

Edit `.env` with your local settings. At minimum you need `JWT_SECRET` and `DB_DSN`. See [Configuration](#configuration) for all variables.

### 4. Build the Binary

```bash
make build
# Output: ./bin/clarity-api
```

Or run directly without building:

```bash
make run
```

### 5. Verify the Server is Running

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

---

## Configuration

All configuration is driven by environment variables. No config files are required. The `.env.example` file documents every variable.

### Required Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `JWT_SECRET` | HS256 signing secret — must be at least 32 characters | `a-very-long-random-secret-key-here` |
| `DB_DSN` | Database connection string | See [Database](#database) |

### Optional Variables (with defaults)

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_ENV` | `development` | Environment name (`development`, `staging`, `production`) |
| `APP_PORT` | `8080` | HTTP listen port |
| `APP_BASE_URL` | `http://localhost:8080` | Public base URL used in generated links |
| `DB_DRIVER` | `postgres` | Database driver (`postgres` or `mysql`) |
| `JWT_ACCESS_TTL_MINUTES` | `15` | Access token lifetime in minutes |
| `JWT_REFRESH_TTL_DAYS` | `7` | Refresh token lifetime in days |
| `REDIS_ADDR` | `localhost:6379` | Redis address for caching |
| `REDIS_PASSWORD` | _(empty)_ | Redis password if auth is enabled |
| `S3_ENDPOINT` | _(empty)_ | S3 or MinIO endpoint URL |
| `S3_BUCKET` | `clarity-documents` | Bucket name for document storage |
| `S3_ACCESS_KEY` | _(empty)_ | S3 access key ID |
| `S3_SECRET_KEY` | _(empty)_ | S3 secret access key |
| `S3_REGION` | `eu-west-1` | S3 region |
| `S3_PRESIGN_TTL_MINUTES` | `15` | Presigned URL expiry |
| `RATE_LIMIT_REQUESTS` | `100` | Max requests per window per tenant |
| `RATE_LIMIT_WINDOW_SECONDS` | `60` | Rate limit window duration |
| `IDEMPOTENCY_TTL_HOURS` | `24` | How long idempotency keys are cached |

---

## Database

The API supports **PostgreSQL 15+** and **MariaDB 11.4 LTS**. The schema was designed for SQL Server 2022 but the Go driver layer uses `lib/pq` (PostgreSQL-compatible) by default.

### PostgreSQL DSN

```
DB_DRIVER=postgres
DB_DSN=postgres://clarity_user:secret@localhost:5432/clarity?sslmode=disable
```

### MariaDB / MySQL DSN

```
DB_DRIVER=mysql
DB_DSN=clarity_user:secret@tcp(localhost:3306)/clarity?parseTime=true&loc=UTC&charset=utf8mb4
```

> **Note:** If using MySQL/MariaDB you will need to add the `github.com/go-sql-driver/mysql` driver to `go.mod` and import it in `internal/db/db.go`. See the TODO comment in that file.

### Running PostgreSQL Locally with Docker

```bash
docker run -d \
  --name clarity-pg \
  -e POSTGRES_USER=clarity_user \
  -e POSTGRES_PASSWORD=secret \
  -e POSTGRES_DB=clarity \
  -p 5432:5432 \
  postgres:15-alpine
```

### Running MariaDB Locally with Docker

```bash
docker run -d \
  --name clarity-mariadb \
  -e MARIADB_USER=clarity_user \
  -e MARIADB_PASSWORD=secret \
  -e MARIADB_DATABASE=clarity \
  -e MARIADB_ROOT_PASSWORD=root \
  -p 3306:3306 \
  mariadb:11.4
```

---

## Project Structure

```
clarity-api/
│
├── cmd/
│   └── api/
│       └── main.go              # Entry point — wires config, DB, router, graceful shutdown
│
├── internal/
│   ├── apierr/
│   │   └── errors.go            # Typed API errors (BadRequest, Forbidden, NotFound, etc.)
│   │                            # All service and repository errors should return *apierr.APIError
│   │
│   ├── audit/
│   │   └── audit.go             # Append-only audit log writer
│   │                            # Call audit.Logger.Write() inside every mutating transaction
│   │
│   ├── claims/
│   │   └── claims.go            # JWT claims extraction from context
│   │                            # Use claims.FromCtx(ctx) in handlers and services
│   │
│   ├── config/
│   │   └── config.go            # Typed configuration loaded from environment variables
│   │
│   ├── ctxkeys/
│   │   └── keys.go              # Context key constants (avoids string collisions)
│   │
│   ├── db/
│   │   └── db.go                # sql.DB wrapper with WithTx(fn) helper for transactions
│   │
│   ├── jwtutil/
│   │   └── jwtutil.go           # JWT sign / parse / NewAccessClaims / NewRefreshClaims
│   │
│   ├── middleware/
│   │   ├── auth.go              # JWT Bearer token validation → injects Claims into context
│   │   ├── context.go           # context.WithValue helper
│   │   ├── idempotency.go       # Idempotency-Key response caching
│   │   ├── logger.go            # Structured per-request logging via slog
│   │   └── ratelimit.go         # Per-tenant rate limiting (pluggable RateLimiter interface)
│   │
│   ├── pagination/
│   │   └── pagination.go        # Parses ?page=&per_page= query params (max 100)
│   │
│   ├── response/
│   │   └── response.go          # JSON response helpers: OK, Created, PageOf, Error, NoContent
│   │
│   ├── router/
│   │   └── router.go            # Full chi router — all 141 routes mounted and wired
│   │
│   └── domain/                  # 27 domain packages — one per bounded context
│       ├── auth/
│       ├── users/
│       ├── delegations/
│       ├── departments/
│       ├── costcentres/
│       ├── locations/
│       ├── currencies/
│       ├── fxrates/
│       ├── costcategories/
│       ├── smtypes/
│       ├── rejectionreasons/
│       ├── vendors/
│       ├── budgetperiods/
│       ├── budgets/
│       ├── budgetlines/
│       ├── agreements/
│       ├── intakerequests/
│       ├── approvalworkflow/
│       ├── purchaseorders/
│       ├── actuals/
│       ├── forecasts/
│       ├── periodclose/
│       ├── auditlog/
│       ├── notifications/
│       ├── aijustification/
│       ├── admin/
│       └── documents/
│
├── go.mod
├── go.sum
├── Makefile
├── .env.example
└── README.md
```

---

## Domain Package Structure

Every domain follows the same four-file layout:

```
internal/domain/<domain>/
├── model.go        # Data structs, CreateRequest, UpdateRequest
├── repository.go   # Repository interface + SQL implementation (data access only)
├── service.go      # Service interface + business logic implementation
└── handler.go      # HTTP handlers — decode request, call service, write response
```

### Architecture Flow

```
HTTP Request
    │
    ▼
middleware/auth.go       ← validates JWT, injects claims into context
    │
    ▼
domain/handler.go        ← decodes JSON body, extracts URL params and claims
    │
    ▼
domain/service.go        ← enforces business rules, role checks, calls repository
    │
    ▼
domain/repository.go     ← executes SQL, returns typed structs or errors
    │
    ▼
internal/db/db.go        ← *sql.DB wrapper
```

### Dependency Rule

> Dependencies only point **inward**. Handlers depend on services. Services depend on repositories. Repositories depend on `*db.DB`. Nothing in `domain/` imports from `router/` or other domain packages.

---

## Implementing a Domain

All domain packages compile and have their interfaces defined. The SQL bodies are stubbed with `// TODO` comments. Here is how to implement one.

### Step 1 — Fill in the Model

Open `internal/domain/<domain>/model.go` and add the real struct fields that match your database columns:

```go
// Before (stub)
type Budget struct {
    ID        string    `json:"id"         db:"id"`
    TenantID  string    `json:"-"          db:"tenant_id"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
    // TODO: add domain-specific fields
}

// After (implemented)
type Budget struct {
    ID           string    `json:"id"            db:"id"`
    TenantID     string    `json:"-"             db:"tenant_id"`
    Name         string    `json:"name"          db:"name"`
    DepartmentID string    `json:"department_id" db:"department_id"`
    PeriodID     string    `json:"period_id"     db:"period_id"`
    Status       string    `json:"status"        db:"status"`
    BudgetType   string    `json:"budget_type"   db:"budget_type"`
    Currency     string    `json:"currency"      db:"currency"`
    CreatedBy    string    `json:"created_by"    db:"created_by_user_id"`
    CreatedAt    time.Time `json:"created_at"    db:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"    db:"updated_at"`
}
```

Also fill in `CreateRequest` and `UpdateRequest` with the fields that arrive in POST/PUT request bodies.

### Step 2 — Implement Repository SQL

Open `internal/domain/<domain>/repository.go`. Each method has a `// TODO` comment describing what query to write:

```go
func (r *repository) List(ctx context.Context, tenantID string, page, perPage int) ([]Budget, int, error) {
    // TODO: SELECT ... FROM budgets WHERE tenant_id=$1 LIMIT $2 OFFSET $3

    rows, err := r.db.QueryContext(ctx,
        `SELECT id, tenant_id, name, department_id, period_id, status, budget_type,
                currency, created_by_user_id, created_at, updated_at
         FROM budgets
         WHERE tenant_id = $1
         ORDER BY created_at DESC
         LIMIT $2 OFFSET $3`,
        tenantID, perPage, (page-1)*perPage,
    )
    if err != nil {
        return nil, 0, fmt.Errorf("budgets.List: %w", err)
    }
    defer rows.Close()

    var results []Budget
    for rows.Next() {
        var b Budget
        if err := rows.Scan(&b.ID, &b.TenantID, &b.Name, &b.DepartmentID,
            &b.PeriodID, &b.Status, &b.BudgetType, &b.Currency,
            &b.CreatedBy, &b.CreatedAt, &b.UpdatedAt); err != nil {
            return nil, 0, fmt.Errorf("budgets.List scan: %w", err)
        }
        results = append(results, b)
    }

    var total int
    r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM budgets WHERE tenant_id=$1`, tenantID).Scan(&total)
    return results, total, nil
}
```

### Step 3 — Add Business Logic to Service

Open `internal/domain/<domain>/service.go`. Services are where you enforce rules before calling the repository:

```go
func (s *service) Create(ctx context.Context, tenantID, userID string, req CreateRequest) (*Budget, error) {
    // 1. Validate the period is open
    // 2. Validate the department belongs to this tenant
    // 3. Check for duplicate budget (same dept + period)
    // 4. Call repo.Create
    // 5. Write audit log
    return s.repo.Create(ctx, tenantID, req)
}
```

### Step 4 — Wire Domain-Specific Handler Actions

For actions beyond basic CRUD (e.g., `Submit`, `Approve`, `Reject`), handler stubs are already in `handler.go` with a `http.StatusNotImplemented` response. Replace them with real logic:

```go
// Submit handles POST /budgets/{budgetId}/submit
func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
    c, err := claims.FromCtx(r.Context())
    if err != nil {
        response.Error(w, apierr.Unauthorized("missing claims"))
        return
    }
    budgetID := chi.URLParam(r, "budgetId")
    if err := h.svc.Submit(r.Context(), c.TenantID, c.Subject, budgetID); err != nil {
        response.Error(w, err)
        return
    }
    response.NoContent(w)
}
```

---

## Make Targets

```bash
make run          # Run the API server (go run ./cmd/api/.)
make build        # Build the binary to ./bin/clarity-api
make test         # Run all tests with race detection and coverage
make lint         # Run golangci-lint
make tidy         # Run go mod tidy
make docker-build # Build the Docker image
make docker-run   # Run the Docker container with .env
```

---

## Running Tests

```bash
make test
```

Tests are located alongside the code they test. Use `_test.go` suffix and the same package name for white-box testing, or `_test` suffix for black-box tests.

To run tests for a single domain:

```bash
go test ./internal/domain/budgets/... -v -cover
```

---

## API Endpoints

All 141 routes are mounted in `internal/router/router.go`. Public routes (no JWT required) are:

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/auth/login` | Authenticate and obtain JWT tokens |
| `POST` | `/v1/auth/refresh` | Exchange a refresh token |
| `POST` | `/v1/auth/password/reset-request` | Request a password reset email |
| `POST` | `/v1/auth/password/reset` | Complete a password reset |
| `POST` | `/v1/auth/mfa/verify` | Complete an MFA challenge |
| `GET`  | `/v1/currencies` | List all currencies (no auth) |
| `GET`  | `/health` | Health check |

All other routes require a valid `Authorization: Bearer <access_token>` header. The full endpoint reference is documented in the [API Endpoint Developer Reference](../clarity-api-endpoint-reference.docx).

---

## Authentication Flow

```
1. POST /v1/auth/login  →  { access_token, refresh_token }
                            OR { mfa_required: true, mfa_token }

2. If MFA required:
   POST /v1/auth/mfa/verify  →  { access_token, refresh_token }

3. Include in all requests:
   Authorization: Bearer <access_token>

4. When access_token expires (15 min):
   POST /v1/auth/refresh  →  { access_token, refresh_token }
   (Old refresh token is rotated — single use)
```

---

## Error Response Format

All errors follow the same envelope structure:

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "budget not found",
    "details": []
  }
}
```

Common error codes: `BAD_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `CONFLICT`, `UNPROCESSABLE_ENTITY`, `TOO_MANY_REQUESTS`, `INTERNAL_SERVER_ERROR`, `ACCOUNT_LOCKED`, `MFA_REQUIRED`.

---

## Pagination

All list endpoints accept `?page=1&per_page=25`. The response envelope includes:

```json
{
  "data": [...],
  "pagination": {
    "page": 1,
    "per_page": 25,
    "total": 143,
    "total_pages": 6
  }
}
```

Maximum `per_page` is 100. Default is 25.

---

## Rate Limiting

Every authenticated request is evaluated against the per-tenant rate limit. Headers are returned on every response:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 87
X-RateLimit-Reset: 1740000000
```

When exceeded, the API returns `429 Too Many Requests` with a `Retry-After` header.

The `RateLimiter` interface in `middleware/ratelimit.go` must be implemented. Wire in a Redis-backed implementation for production. An in-memory implementation is suitable for development.

---

## Idempotency

For POST, PUT, and DELETE requests, include an `Idempotency-Key: <uuid>` header. If the same key is sent again within 24 hours, the original response is returned without re-executing the handler. The response will include `X-Idempotency-Replayed: true`.

The `IdempotencyStore` interface in `middleware/idempotency.go` must be backed by Redis in production.

---

## Audit Logging

Every mutating operation must write to `audit_log`. The `audit.Logger` in `internal/audit/audit.go` is the sole write path. Always call it inside the same database transaction as the operation being audited:

```go
return db.WithTx(func(tx *sql.Tx) error {
    // 1. Execute the main operation
    if _, err := tx.ExecContext(ctx, `UPDATE budgets SET status=$1 WHERE id=$2`, ...); err != nil {
        return err
    }
    // 2. Write audit entry in the same transaction
    return auditLogger.Write(ctx, tx, audit.Entry{
        EntityType:  "budgets",
        EntityID:    budgetID,
        Action:      audit.ActionUpdate,
        BeforeState: audit.Snapshot(before),
        AfterState:  audit.Snapshot(after),
    })
})
```

---

## Multi-Tenancy

Every query **must** include `WHERE tenant_id = $1`. The tenant ID is extracted from the JWT `tid` claim and is never accepted from the request body or URL. The `claims.TenantID(ctx)` helper provides it anywhere a context is available.

---

## Contributing

1. Branch from `main` using the pattern `feat/<domain>/<description>` or `fix/<description>`
2. Implement the domain following the four-step guide above
3. Write tests for the service layer (business logic) at minimum
4. Run `make lint` and `make test` before opening a pull request
5. Ensure `go build ./...` passes with no warnings

---

## Licence

Internal — Computershare / Clarity Platform. Not for external distribution.
