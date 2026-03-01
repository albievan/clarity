# Clarity API — Go Framework

**Budget Management Platform · Internal REST API**
`github.com/albievan/clarity/clarity-api`

---

## Overview

This repository contains the Go API framework for the **Clarity Budget Management Platform**. It implements a hexagonal (ports and adapters) architecture across 27 domain packages, covering all 141 endpoints defined in the Clarity OpenAPI specification.

The framework separates concerns cleanly so that developers can implement SQL queries, business rules, and HTTP handling independently of each other.

---

## Prerequisites

### Required

| Tool | Minimum Version | Purpose | Install |
|------|----------------|---------|---------|
| **Go** | 1.22 | Compiler and toolchain | https://go.dev/dl/ |
| **Git** | 2.x | Source control | https://git-scm.com/ |
| **SQL Server** or **MariaDB** | 2019+ / 10.6+ | Primary database | See [Database](#database) |

### Recommended for Development

| Tool | Purpose | Install |
|------|---------|---------| 
| **Docker** + **Docker Compose** | Run DB and Redis locally without installing them | https://docs.docker.com/get-docker/ |
| **Redis** | Session caching, rate limiting, idempotency store | https://redis.io/download |
| **golangci-lint** | Static analysis and linting | https://golangci-lint.run/usage/install/ |
| **MinIO** | Local S3-compatible object storage for document uploads | https://min.io/download |
| **ngrok** | HTTPS tunnel needed for Apple OAuth callbacks in development | https://ngrok.com/ |

---

## Getting Started

### 1. Clone and Install Dependencies

```bash
git clone https://github.com/albievan/clarity/clarity-api.git
cd clarity-api
go mod tidy
```

`go mod tidy` downloads all transitive dependencies including `golang.org/x/crypto` (bcrypt) and the database drivers.

### 2. Configure Environment Variables

```bash
cp .env.example .env
```

Edit `.env`. At minimum you need `JWT_SECRET` and `DB_DSN`. See [Configuration](#configuration) for all variables.

### 3. Run the Server

```bash
make run
```

### 4. Verify

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

---

## Passing the .env File to the Application

The application uses `godotenv` to automatically load a `.env` file from the working directory at startup. No extra flags are needed.

```bash
make run      # auto-loads .env from project root
make start    # builds binary then runs it — also auto-loads .env
```

**How it works:** `godotenv.Load()` is called at the top of `main()`. It reads `.env` and sets each key as an environment variable — but only if that variable is **not already set**. This means:

- **Development:** `.env` is loaded automatically.
- **Production:** real environment variables injected by your platform take precedence; `.env` is ignored even if present.

---

## Configuration

All configuration is driven by environment variables. The `.env.example` file documents every variable. Below is a summary.

### Core

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_ENV` | `development` | `development`, `staging`, or `production` |
| `APP_PORT` | `8080` | HTTP listen port |
| `APP_BASE_URL` | `http://localhost:8080` | Public API base URL — used in OAuth redirect URI defaults |
| `FRONTEND_URL` | `http://localhost:3000` | Frontend app URL — OAuth callbacks redirect here after login |
| `CORS_ALLOWED_ORIGINS` | _(empty)_ | Extra comma-separated origins to allow. `FRONTEND_URL` and `APP_BASE_URL` are always included. In `development` mode the `null` origin (file:// loads) is also included automatically. |

### Database

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_DRIVER` | _(required)_ | `sqlserver` or `mysql` |
| `DB_DSN` | _(required)_ | Connection string — see [Database](#database) |

### JWT

| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_SECRET` | _(required)_ | HS256 signing secret — minimum 32 characters |
| `JWT_ACCESS_TTL_MINUTES` | `15` | Access token lifetime |
| `JWT_REFRESH_TTL_DAYS` | `7` | Refresh token lifetime |

### OAuth — Google

| Variable | Default | Description |
|----------|---------|-------------|
| `GOOGLE_CLIENT_ID` | _(empty — disables Google login)_ | OAuth 2.0 client ID from Google Cloud Console |
| `GOOGLE_CLIENT_SECRET` | _(empty)_ | OAuth 2.0 client secret |
| `GOOGLE_REDIRECT_URL` | `{APP_BASE_URL}/v1/auth/oauth/google/callback` | Overrides the default callback URL |

### OAuth — Apple

| Variable | Default | Description |
|----------|---------|-------------|
| `APPLE_CLIENT_ID` | _(empty — disables Apple login)_ | Apple Services ID (e.g. `com.clarity.api`) |
| `APPLE_CLIENT_SECRET` | _(empty)_ | Pre-generated ES256-signed JWT — see note below |
| `APPLE_REDIRECT_URL` | `{APP_BASE_URL}/v1/auth/oauth/apple/callback` | Must be HTTPS in production; use ngrok in dev |

### Other Services

| Variable | Default | Description |
|----------|---------|-------------|
| `REDIS_ADDR` | `localhost:6379` | Redis for caching, rate limits, idempotency |
| `REDIS_PASSWORD` | _(empty)_ | Redis auth password |
| `S3_BUCKET` | `clarity-documents` | Object storage bucket |
| `S3_ACCESS_KEY` / `S3_SECRET_KEY` | _(empty)_ | S3 credentials |
| `S3_REGION` | `eu-west-1` | S3 region |
| `RATE_LIMIT_REQUESTS` | `100` | Max requests per window per tenant |
| `RATE_LIMIT_WINDOW_SECONDS` | `60` | Rate limit window |
| `IDEMPOTENCY_TTL_HOURS` | `24` | Idempotency key cache lifetime |

---

## Authentication

The API supports three login providers. All three issue the same JWT pair (`access_token` + `refresh_token`) so downstream code is provider-agnostic.

### Local Users (email + password)

```
POST /v1/auth/login
Body: { "tenant_id": "...", "email": "...", "password": "..." }

Response: { "access_token": "...", "refresh_token": "..." }
      or: { "mfa_required": true, "mfa_token": "..." }  ← if MFA is enabled
```

Local users are created via `POST /v1/users` (admin only). Passwords are hashed using the `PasswordHasher` interface in `internal/domain/auth/service.go`.

**Password security:**

- Development: the default stub uses SHA-256 — adequate for testing, **not for production**.
- Production: replace `stubHasher` with `BcryptHasher` after `go mod tidy` resolves `golang.org/x/crypto/bcrypt`. The replacements are documented with inline comments in `service.go`.

**Account lockout** is controlled by the `security_policy` table. Defaults when no row exists:

| Policy | Default |
|--------|---------|
| Failed login threshold | 5 attempts |
| Lockout duration | 30 minutes |
| Minimum password length | 12 characters |
| MFA required | false |

Manage per-tenant policy via `PUT /v1/admin/security-policy`.

**MFA (TOTP):**

1. `POST /v1/auth/mfa/setup` — generates a secret and an `otpauth://` URI for QR code
2. `POST /v1/auth/mfa/confirm` — validates first TOTP code, activates MFA, returns 8 backup codes
3. On login, when `mfa_required: true` is returned, submit the TOTP code to `POST /v1/auth/mfa/verify`

The TOTP provider uses a stub in development (accepts any 6-digit code). Replace with `TOTPImpl` backed by `github.com/pquerna/otp` for production — again documented inline in `service.go`.

### Google OAuth 2.0

The flow is a standard browser-redirect OAuth2 authorisation code flow.

**Setup:**
1. Create an OAuth 2.0 credential in [Google Cloud Console](https://console.cloud.google.com/) → APIs & Services → Credentials
2. Add the redirect URI to "Authorised redirect URIs":
   - Development: `http://localhost:8080/v1/auth/oauth/google/callback`
   - Production: `https://api.clarity.example.com/v1/auth/oauth/google/callback`
3. Set `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET` in `.env`

**Login flow:**

```
1. Frontend calls:
   GET /v1/auth/oauth/google/init?tenant_id=<id>
   ← returns { "redirect_url": "https://accounts.google.com/o/oauth2/v2/auth?..." }

2. Frontend navigates the browser to that URL.

3. Google redirects back to:
   GET /v1/auth/oauth/google/callback?code=...&state=...

4. API exchanges the code, fetches userinfo, finds or creates the user,
   then redirects the browser to:
   {FRONTEND_URL}/auth/callback#access_token=...&refresh_token=...

5. Frontend reads the tokens from the URL fragment.
```

If a Google account's email matches an existing local user, the Google identity is **linked** to that account automatically. The user can then log in with either method.

### Apple Sign In

Apple uses the same code flow but with two differences:
- Apple sends the callback as an HTTP **POST** (`response_mode=form_post`) rather than a GET.
- The `APPLE_CLIENT_SECRET` is not a static secret — it is a **short-lived ES256-signed JWT** you generate from your Apple `.p8` private key. It expires after a maximum of 6 months. See [Apple's documentation](https://developer.apple.com/documentation/sign_in_with_apple/generate_and_validate_tokens).
- Apple's callback URL **must be HTTPS**. Use [ngrok](https://ngrok.com/) (`ngrok http 8080`) during development and set `APPLE_REDIRECT_URL` to the ngrok HTTPS URL.

**Setup:**
1. In [Apple Developer](https://developer.apple.com/) → Certificates, IDs & Profiles, register a Services ID
2. Enable "Sign In with Apple" and add your domain + redirect URL
3. Create a Key with "Sign In with Apple" capability, download the `.p8` file
4. Generate the `client_secret` JWT and set `APPLE_CLIENT_ID` + `APPLE_CLIENT_SECRET` in `.env`

**Login flow** is identical to Google — replace `google` with `apple` in the endpoint paths.

### Full Auth Flow Reference

```
── Local login ──────────────────────────────────────────────────────────────

POST /v1/auth/login
  → 200 { access_token, refresh_token }          normal login
  → 200 { mfa_required: true, mfa_token }        MFA enabled

POST /v1/auth/mfa/verify { mfa_token, code }
  → 200 { access_token, refresh_token }

── OAuth login ──────────────────────────────────────────────────────────────

GET  /v1/auth/oauth/google/init?tenant_id=…      → redirect_url
GET  /v1/auth/oauth/google/callback              (handled by browser)
GET  /v1/auth/oauth/apple/init?tenant_id=…       → redirect_url
POST /v1/auth/oauth/apple/callback               (Apple POST callback)

── Token management ─────────────────────────────────────────────────────────

POST /v1/auth/refresh { refresh_token }
  → 200 { access_token, refresh_token }          (token rotated)

POST /v1/auth/logout                             revokes current session

── All authenticated requests ───────────────────────────────────────────────

Authorization: Bearer <access_token>

── MFA management ───────────────────────────────────────────────────────────

POST /v1/auth/mfa/setup
POST /v1/auth/mfa/confirm { code }
POST /v1/auth/mfa/disable { code }

── Password ─────────────────────────────────────────────────────────────────

POST /v1/auth/password/change { current_password, new_password }
POST /v1/auth/password/reset-request { tenant_id, email }    (always 204)
POST /v1/auth/password/reset { token, new_password }

── Sessions ─────────────────────────────────────────────────────────────────

GET    /v1/auth/sessions
DELETE /v1/auth/sessions/{sessionId}
```

---

## User Management

User management is handled by the `users` domain. All endpoints require authentication; most require `it_admin` or `finance_controller` role.

### Endpoints

| Method | Path | Description | Roles Required |
|--------|------|-------------|----------------|
| `GET` | `/v1/users` | List / search users | admin |
| `POST` | `/v1/users` | Create a local user | admin |
| `GET` | `/v1/users/{userId}` | Get a single user | admin or self |
| `PUT` | `/v1/users/{userId}` | Update name fields | admin or self |
| `DELETE` | `/v1/users/{userId}` | Deprovision (soft delete) | admin |
| `POST` | `/v1/users/{userId}/lock` | Lock account | admin |
| `POST` | `/v1/users/{userId}/unlock` | Unlock account | admin |
| `GET` | `/v1/users/{userId}/roles` | List role assignments | admin or self |
| `POST` | `/v1/users/{userId}/roles` | Assign a role | admin |
| `DELETE` | `/v1/users/{userId}/roles/{assignmentId}` | Revoke a role | admin |
| `GET` | `/v1/users/{userId}/identities` | List linked OAuth identities | admin or self |
| `DELETE` | `/v1/users/{userId}/identities/{identityId}` | Unlink an OAuth identity | admin or self |

### Creating a Local User

```http
POST /v1/users
Authorization: Bearer <admin_access_token>

{
  "email": "alice@example.com",
  "first_name": "Alice",
  "last_name": "Smith",
  "password": "securepassword123",
  "roles": ["budget_owner"]
}
```

Password must be at least 12 characters. If `roles` is omitted, the user is assigned `budget_requestor` by default.

### Search and Filtering

```
GET /v1/users?search=alice&status=active&auth_provider=google&role=budget_owner&page=1&per_page=25
```

| Parameter | Description |
|-----------|-------------|
| `search` | Searches email, first name, last name, and display name (LIKE) |
| `status` | `active`, `locked`, or `deprovisioned` |
| `auth_provider` | `local`, `google`, or `apple` |
| `role` | Return only users with this role assigned |
| `page` / `per_page` | Pagination (default: 1 / 25, max per_page: 100) |

### Available Roles

| Role | Description |
|------|-------------|
| `it_admin` | Full platform administration — user management, tenant config |
| `finance_controller` | Financial oversight — audit log, all budget visibility |
| `budget_owner` | Owns department budgets — approve / submit |
| `budget_approver` | Approves budget submissions |
| `dept_head` | Department head — visibility into department budgets |
| `budget_requestor` | Default role — creates and submits budget requests |

### OAuth Identity Linking

When a user signs in via Google or Apple for the first time, the API checks whether an account with that email already exists:

- **Match found** — the OAuth identity is linked to the existing account. The user can now log in with either method.
- **No match** — a new user account is provisioned automatically with `budget_requestor` role and `auth_provider` set to the OAuth provider.

To view which OAuth providers a user has linked, call `GET /v1/users/{userId}/identities`. To remove a linked identity, call `DELETE /v1/users/{userId}/identities/{identityId}`. Unlinking is blocked if it would leave the user with no way to log in (i.e. no local password and no remaining OAuth identities).

---

## Database

Both SQL Server and MariaDB are supported. Set `DB_DRIVER` to `sqlserver` or `mysql`.

### DSN Format

```bash
# SQL Server
DB_DRIVER=sqlserver
DB_DSN=sqlserver://clarity_user:YourPassword@localhost:1433?database=clarity

# MariaDB / MySQL
DB_DRIVER=mysql
DB_DSN=clarity_user:YourPassword@tcp(localhost:3306)/clarity?parseTime=true&loc=UTC&charset=utf8mb4
```

### Running SQL Server Locally

```bash
docker run -d \
  --name clarity-mssql \
  -e ACCEPT_EULA=Y \
  -e MSSQL_SA_PASSWORD=YourStr0ngPassword \
  -p 1433:1433 \
  mcr.microsoft.com/mssql/server:2022-latest
```

### Required Schema Changes for Auth and Users

The following tables must exist in addition to any tables already required by other domains. MariaDB DDL is shown; adjust `BIT` to `TINYINT(1)` and `VARCHAR` lengths as needed for SQL Server.

```sql
-- Additional columns required on the existing users table
ALTER TABLE users
  ADD COLUMN password_hash        VARCHAR(255) NULL,
  ADD COLUMN status               VARCHAR(20)  NOT NULL DEFAULT 'active',
  ADD COLUMN auth_provider        VARCHAR(20)  NOT NULL DEFAULT 'local',
  ADD COLUMN avatar_url           VARCHAR(500) NULL,
  ADD COLUMN first_name           VARCHAR(100) NULL,
  ADD COLUMN last_name            VARCHAR(100) NULL,
  ADD COLUMN display_name         VARCHAR(200) NULL,
  ADD COLUMN failed_login_count   INT          NOT NULL DEFAULT 0,
  ADD COLUMN locked_until         DATETIME     NULL,
  ADD COLUMN mfa_enabled          TINYINT      NOT NULL DEFAULT 0,
  ADD COLUMN mfa_secret           VARCHAR(100) NULL,
  ADD COLUMN mfa_secret_pending   VARCHAR(100) NULL,
  ADD COLUMN require_pw_change    TINYINT      NOT NULL DEFAULT 0,
  ADD COLUMN last_login_at        DATETIME     NULL;

-- OAuth identities — one row per linked social account
CREATE TABLE oauth_identities (
  id           VARCHAR(32)  NOT NULL PRIMARY KEY,
  user_id      VARCHAR(32)  NOT NULL,
  tenant_id    VARCHAR(32)  NOT NULL,
  provider     VARCHAR(20)  NOT NULL,    -- 'google' | 'apple'
  provider_uid VARCHAR(255) NOT NULL,   -- subject claim from the provider (stable unique ID)
  email        VARCHAR(255) NOT NULL,
  display_name VARCHAR(200) NULL,
  avatar_url   VARCHAR(500) NULL,
  created_at   DATETIME     NOT NULL,
  UNIQUE KEY uq_identity (tenant_id, provider, provider_uid),
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Role assignments — many-to-many between users and roles
CREATE TABLE user_roles (
  id          VARCHAR(32)  NOT NULL PRIMARY KEY,
  user_id     VARCHAR(32)  NOT NULL,
  tenant_id   VARCHAR(32)  NOT NULL,
  role_name   VARCHAR(50)  NOT NULL,
  granted_by  VARCHAR(32)  NOT NULL,
  granted_at  DATETIME     NOT NULL,
  UNIQUE KEY uq_user_role (tenant_id, user_id, role_name),
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Sessions — one row per active refresh token
CREATE TABLE sessions (
  id           VARCHAR(32)  NOT NULL PRIMARY KEY,
  user_id      VARCHAR(32)  NOT NULL,
  tenant_id    VARCHAR(32)  NOT NULL,
  token_hash   VARCHAR(64)  NOT NULL,   -- SHA-256 hex of raw refresh token
  ip_address   VARCHAR(45)  NULL,
  user_agent   VARCHAR(500) NULL,
  expires_at   DATETIME     NOT NULL,
  revoked_at   DATETIME     NULL,
  last_used_at DATETIME     NOT NULL,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Password reset tokens
CREATE TABLE password_reset_tokens (
  id         VARCHAR(32) NOT NULL PRIMARY KEY,
  user_id    VARCHAR(32) NOT NULL,
  tenant_id  VARCHAR(32) NOT NULL,
  token_hash VARCHAR(64) NOT NULL UNIQUE,  -- SHA-256 hex
  expires_at DATETIME    NOT NULL,
  used_at    DATETIME    NULL,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- MFA backup codes (8 per user, single-use)
CREATE TABLE mfa_backup_codes (
  id        VARCHAR(32) NOT NULL PRIMARY KEY,
  user_id   VARCHAR(32) NOT NULL,
  tenant_id VARCHAR(32) NOT NULL,
  code_hash VARCHAR(64) NOT NULL,   -- SHA-256 hex of 10-char plaintext code
  used_at   DATETIME    NULL,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Security policy — one row per tenant, API falls back to safe defaults if absent
CREATE TABLE security_policy (
  tenant_id              VARCHAR(32) NOT NULL PRIMARY KEY,
  lockout_threshold      INT         NOT NULL DEFAULT 5,
  lockout_duration_mins  INT         NOT NULL DEFAULT 30,
  mfa_required           TINYINT     NOT NULL DEFAULT 0,
  min_password_length    INT         NOT NULL DEFAULT 12,
  password_history_count INT         NOT NULL DEFAULT 5
);
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
│   │
│   ├── audit/
│   │   └── audit.go             # Append-only audit log writer — call inside DB transactions
│   │
│   ├── claims/
│   │   └── claims.go            # JWT claims extraction from context
│   │
│   ├── config/
│   │   └── config.go            # Typed config loaded from env — includes OAuthConfig
│   │
│   ├── db/
│   │   └── db.go                # sql.DB wrapper with WithTx(fn) transaction helper
│   │
│   ├── jwtutil/
│   │   └── jwtutil.go           # JWT sign / parse / NewAccessClaims / NewRefreshClaims
│   │
│   ├── middleware/
│   │   ├── auth.go              # JWT Bearer validation — injects Claims into context
│   │   ├── idempotency.go       # Idempotency-Key response caching
│   │   ├── logger.go            # Structured per-request logging via slog
│   │   └── ratelimit.go         # Per-tenant rate limiting
│   │
│   ├── pagination/
│   │   └── pagination.go        # Parses ?page=&per_page= (max 100)
│   │
│   ├── response/
│   │   └── response.go          # JSON helpers: OK, Created, PageOf, Error, NoContent
│   │
│   ├── router/
│   │   └── router.go            # All routes mounted — auth, OAuth, users, 27 domains
│   │
│   └── domain/
│       ├── auth/
│       │   ├── model.go         # Auth request/response types, session, MFA, password reset
│       │   ├── repository.go    # SQL: users, sessions, MFA, password reset, security policy
│       │   ├── service.go       # Login, logout, refresh, MFA, password, sessions
│       │   ├── handler.go       # HTTP handlers for all auth endpoints
│       │   └── oauth.go         # Google + Apple OAuth2 flows — init, callback, state store
│       │
│       ├── users/
│       │   ├── model.go         # User, OAuthIdentity, RoleAssignment, Filter, request types
│       │   ├── repository.go    # SQL: CRUD, search, roles (user_roles), OAuth identities
│       │   ├── service.go       # Business rules, FindOrCreateOAuthUser, role enforcement
│       │   └── handler.go       # List, create, get, update, deprovision, lock, unlock,
│       │                        #   roles, identities
│       │
│       └── [24 other domains]   # budgets, budgetlines, agreements, actuals, etc.
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
├── model.go        # Data structs, request/response types
├── repository.go   # Repository interface + SQL implementation
├── service.go      # Service interface + business logic
└── handler.go      # HTTP handlers
```

### Architecture Flow

```
HTTP Request
    │
    ▼
middleware/auth.go      ← validates JWT Bearer token, injects claims into context
    │
    ▼
domain/handler.go       ← decodes request, extracts URL params and claims
    │
    ▼
domain/service.go       ← enforces business rules and role checks
    │
    ▼
domain/repository.go    ← executes SQL, returns typed structs or errors
    │
    ▼
internal/db/db.go       ← *sql.DB wrapper
```

**Dependency rule:** dependencies only point inward. Handlers depend on services; services depend on repositories; repositories depend on `*db.DB`. The `users` domain is the only exception — it is imported by the `auth` domain to support `FindOrCreateOAuthUser` during OAuth callbacks.

---

## Implementing a Domain

All domain packages compile with their interfaces defined. SQL bodies are stubbed with `// TODO` comments.

### Step 1 — Fill in the Model

Add real struct fields matching your database columns to `model.go`, then fill in `CreateRequest` and `UpdateRequest` with the fields that arrive in POST/PUT request bodies.

### Step 2 — Implement Repository SQL

Replace `// TODO` stubs in `repository.go`. Use `?` placeholders — the same syntax works for both the `mysql` and `sqlserver` drivers. Note: `go-mssqldb` maps `?` to `@p1, @p2` automatically.

### Step 3 — Add Business Logic to Service

Services enforce rules before calling the repository: validate inputs, check roles, write audit log entries, coordinate across multiple repo calls.

### Step 4 — Wire Domain-Specific Actions

For actions beyond CRUD (`Submit`, `Approve`, `Reject`, etc.), handler stubs already return `501 Not Implemented`. Replace them with real calls to service methods.

---

## API Reference

### Public Endpoints (no JWT required)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/auth/login` | Authenticate with email + password |
| `POST` | `/v1/auth/refresh` | Exchange a refresh token for a new pair |
| `POST` | `/v1/auth/password/reset-request` | Send password reset email (always returns 204) |
| `POST` | `/v1/auth/password/reset` | Complete a password reset |
| `POST` | `/v1/auth/mfa/verify` | Complete an MFA challenge after login |
| `GET`  | `/v1/auth/oauth/google/init` | Start Google OAuth2 flow |
| `GET`  | `/v1/auth/oauth/google/callback` | Google OAuth2 callback (browser redirect) |
| `GET`  | `/v1/auth/oauth/apple/init` | Start Apple Sign In flow |
| `POST` | `/v1/auth/oauth/apple/callback` | Apple Sign In callback (form POST) |
| `GET`  | `/v1/currencies` | List currencies |
| `GET`  | `/health` | Health check |

### Authenticated Endpoints

All other routes require `Authorization: Bearer <access_token>`. The full endpoint list is in `internal/router/router.go` and the [API Endpoint Developer Reference](../clarity-api-endpoint-reference.docx).

---

## Error Response Format

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "user not found",
    "details": []
  }
}
```

Common codes: `BAD_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `CONFLICT`, `UNPROCESSABLE_ENTITY`, `TOO_MANY_REQUESTS`, `INTERNAL_SERVER_ERROR`, `ACCOUNT_LOCKED`.

---

## Pagination

All list endpoints accept `?page=1&per_page=25`. Maximum `per_page` is 100.

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

---

## Audit Logging

Every mutating operation must write to `audit_log` inside the same database transaction as the operation itself:

```go
return db.WithTx(func(tx *sql.Tx) error {
    _, err := tx.ExecContext(ctx, `UPDATE users SET status=? WHERE id=?`, "locked", userID)
    if err != nil { return err }
    return auditLogger.Write(ctx, tx, audit.Entry{
        EntityType: "users",
        EntityID:   userID,
        Action:     audit.ActionLock,
        BeforeState: audit.Snapshot(before),
        AfterState:  audit.Snapshot(after),
    })
})
```

Auth-specific audit actions are pre-defined: `ActionLogin`, `ActionLogout`, `ActionMFAEnable`, `ActionMFADisable`, `ActionPWChange`, `ActionPWReset`.

---

## Multi-Tenancy

Every query **must** include `WHERE tenant_id = ?`. The tenant ID comes from the JWT `tid` claim and is never accepted from the request body or URL. Use `claims.TenantID(ctx)` anywhere a context is available.

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

## Contributing

1. Branch from `main` using `feat/<domain>/<description>` or `fix/<description>`
2. Implement using the four-step guide above
3. Write service-layer tests at minimum
4. Run `make lint` and `make test` before raising a PR
5. Ensure `go build ./...` exits 0 with no warnings

---

## Licence

Internal — Merwe / Clarity Platform. Not for external distribution.
