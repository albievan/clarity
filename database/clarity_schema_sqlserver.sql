-- =============================================================================
--  CLARITY — Budget Management Platform
--  Database Schema — Microsoft SQL Server
--  All IDs: INT IDENTITY(1,1) (auto-increment)
--  Dialect: SQL Server 2019+ / Azure SQL
-- =============================================================================

-- =============================================================================
-- 0. SCHEMAS
-- =============================================================================
IF NOT EXISTS (SELECT 1 FROM sys.schemas WHERE name = 'platform') EXEC('CREATE SCHEMA platform');
IF NOT EXISTS (SELECT 1 FROM sys.schemas WHERE name = 'auth')     EXEC('CREATE SCHEMA auth');
IF NOT EXISTS (SELECT 1 FROM sys.schemas WHERE name = 'ref')      EXEC('CREATE SCHEMA ref');
IF NOT EXISTS (SELECT 1 FROM sys.schemas WHERE name = 'finance')  EXEC('CREATE SCHEMA finance');
IF NOT EXISTS (SELECT 1 FROM sys.schemas WHERE name = 'workflow') EXEC('CREATE SCHEMA workflow');
IF NOT EXISTS (SELECT 1 FROM sys.schemas WHERE name = 'audit')    EXEC('CREATE SCHEMA audit');
GO

-- =============================================================================
-- 1. PLATFORM — TENANCY
-- =============================================================================

CREATE TABLE platform.tenants (
    id              INT             NOT NULL IDENTITY(1,1),
    name            NVARCHAR(200)   NOT NULL,
    slug            NVARCHAR(100)   NOT NULL,           -- URL-safe identifier
    status          NVARCHAR(20)    NOT NULL DEFAULT 'active'
                                    CHECK (status IN ('active','suspended','decommissioned')),
    timezone        NVARCHAR(60)    NOT NULL DEFAULT 'UTC',
    base_currency   NVARCHAR(3)     NOT NULL DEFAULT 'GBP',
    fiscal_year_start_month TINYINT NOT NULL DEFAULT 1  -- 1=Jan, 4=Apr, 7=Jul, 10=Oct
                                    CHECK (fiscal_year_start_month IN (1,4,7,10)),
    max_quote_uploads TINYINT       NOT NULL DEFAULT 3,
    created_at      DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    updated_at      DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_tenants PRIMARY KEY (id),
    CONSTRAINT uq_tenants_slug UNIQUE (slug)
);

CREATE TABLE platform.feature_flags (
    id              INT             NOT NULL IDENTITY(1,1),
    tenant_id       INT             NOT NULL,
    flag_key        NVARCHAR(100)   NOT NULL,
    is_enabled      BIT             NOT NULL DEFAULT 1,
    description     NVARCHAR(500)   NULL,
    updated_by      INT             NULL,               -- user_id (nullable for system flags)
    updated_at      DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_feature_flags PRIMARY KEY (id),
    CONSTRAINT uq_feature_flags_tenant_key UNIQUE (tenant_id, flag_key),
    CONSTRAINT fk_feature_flags_tenant FOREIGN KEY (tenant_id) REFERENCES platform.tenants(id)
);

CREATE TABLE platform.rate_limit_config (
    id                  INT             NOT NULL IDENTITY(1,1),
    tenant_id           INT             NOT NULL,
    endpoint_category   NVARCHAR(80)    NOT NULL,       -- e.g. 'auth', 'standard_read', 'bulk_import'
    per_user_per_minute INT             NOT NULL,
    per_ip_per_minute   INT             NOT NULL,
    burst_limit         INT             NOT NULL,
    updated_by          INT             NULL,
    updated_at          DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_rate_limit_config PRIMARY KEY (id),
    CONSTRAINT uq_rate_limit_category UNIQUE (tenant_id, endpoint_category),
    CONSTRAINT fk_rate_limit_tenant FOREIGN KEY (tenant_id) REFERENCES platform.tenants(id)
);

-- =============================================================================
-- 2. AUTH — IDENTITY & ACCESS
-- =============================================================================

CREATE TABLE auth.roles (
    id              INT             NOT NULL IDENTITY(1,1),
    name            NVARCHAR(80)    NOT NULL,
    description     NVARCHAR(500)   NULL,
    is_system_role  BIT             NOT NULL DEFAULT 1, -- system roles cannot be deleted
    CONSTRAINT pk_roles PRIMARY KEY (id),
    CONSTRAINT uq_roles_name UNIQUE (name)
);

-- Seed system roles
INSERT INTO auth.roles (name, description, is_system_role) VALUES
('IT Administrator',    'Full platform administration',                        1),
('Finance Controller',  'Full finance access, period close, cross-dept reporting', 1),
('Budget Approver',     'Approve/reject budget submissions',                   1),
('Department Head',     'Review and action intake requests from team',         1),
('Budget Owner',        'Create and manage departmental budgets',              1),
('Budget Requestor',    'Submit budget item requests',                         1),
('Read Only',           'Read-only access to permitted budget data',           1);

CREATE TABLE auth.users (
    id                  INT             NOT NULL IDENTITY(1,1),
    tenant_id           INT             NOT NULL,
    email               NVARCHAR(320)   NOT NULL,
    display_name        NVARCHAR(200)   NOT NULL,
    -- Local auth (NULL if SSO-only)
    password_hash       NVARCHAR(255)   NULL,
    -- MFA
    mfa_enabled         BIT             NOT NULL DEFAULT 0,
    mfa_secret_enc      NVARCHAR(500)   NULL,           -- Vault-encrypted TOTP secret
    mfa_backup_codes    NVARCHAR(1000)  NULL,           -- JSON array of hashed backup codes
    -- Account lifecycle
    status              NVARCHAR(30)    NOT NULL DEFAULT 'active'
                                        CHECK (status IN ('active','locked','suspended','deprovisioned','pending_setup')),
    failed_login_count  TINYINT         NOT NULL DEFAULT 0,
    locked_until        DATETIME2(3)    NULL,
    password_changed_at DATETIME2(3)    NULL,
    require_pw_change   BIT             NOT NULL DEFAULT 1, -- force change on first login
    last_login_at       DATETIME2(3)    NULL,
    last_login_ip       NVARCHAR(45)    NULL,
    -- SSO
    sso_provider        NVARCHAR(40)    NULL            -- 'entra_id', 'google', 'apple', NULL
                                        CHECK (sso_provider IN ('entra_id','google','apple',NULL)),
    sso_subject         NVARCHAR(500)   NULL,           -- IdP sub claim
    scim_external_id    NVARCHAR(500)   NULL,           -- SCIM id from IdP
    -- Audit
    created_at          DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    updated_at          DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_users PRIMARY KEY (id),
    CONSTRAINT uq_users_email UNIQUE (tenant_id, email),
    CONSTRAINT fk_users_tenant FOREIGN KEY (tenant_id) REFERENCES platform.tenants(id)
);

CREATE TABLE auth.user_roles (
    id              INT             NOT NULL IDENTITY(1,1),
    user_id         INT             NOT NULL,
    role_id         INT             NOT NULL,
    -- Optional: scope role to a department (NULL = global scope for tenant)
    department_id   INT             NULL,
    assigned_by     INT             NOT NULL,
    assigned_at     DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    revoked_at      DATETIME2(3)    NULL,
    revoked_by      INT             NULL,
    CONSTRAINT pk_user_roles PRIMARY KEY (id),
    CONSTRAINT fk_user_roles_user FOREIGN KEY (user_id) REFERENCES auth.users(id),
    CONSTRAINT fk_user_roles_role FOREIGN KEY (role_id) REFERENCES auth.roles(id),
    CONSTRAINT fk_user_roles_assigned_by FOREIGN KEY (assigned_by) REFERENCES auth.users(id)
);

CREATE TABLE auth.sessions (
    id              INT             NOT NULL IDENTITY(1,1),
    user_id         INT             NOT NULL,
    tenant_id       INT             NOT NULL,
    token_hash      NVARCHAR(255)   NOT NULL,           -- SHA-256 hash of session token
    ip_address      NVARCHAR(45)    NOT NULL,
    user_agent      NVARCHAR(500)   NULL,
    created_at      DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    expires_at      DATETIME2(3)    NOT NULL,
    revoked_at      DATETIME2(3)    NULL,
    revoke_reason   NVARCHAR(100)   NULL,               -- 'logout','timeout','admin','scim_deprovision'
    CONSTRAINT pk_sessions PRIMARY KEY (id),
    CONSTRAINT uq_sessions_token UNIQUE (token_hash),
    CONSTRAINT fk_sessions_user FOREIGN KEY (user_id) REFERENCES auth.users(id),
    CONSTRAINT fk_sessions_tenant FOREIGN KEY (tenant_id) REFERENCES platform.tenants(id)
);

CREATE TABLE auth.password_history (
    id              INT             NOT NULL IDENTITY(1,1),
    user_id         INT             NOT NULL,
    password_hash   NVARCHAR(255)   NOT NULL,
    changed_at      DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_password_history PRIMARY KEY (id),
    CONSTRAINT fk_pw_history_user FOREIGN KEY (user_id) REFERENCES auth.users(id)
);

CREATE TABLE auth.security_policy (
    id              INT             NOT NULL IDENTITY(1,1),
    tenant_id       INT             NOT NULL,
    setting_key     NVARCHAR(100)   NOT NULL,
    setting_value   NVARCHAR(500)   NOT NULL,
    updated_by      INT             NOT NULL,
    updated_at      DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_security_policy PRIMARY KEY (id),
    CONSTRAINT uq_security_policy UNIQUE (tenant_id, setting_key),
    CONSTRAINT fk_security_policy_tenant FOREIGN KEY (tenant_id) REFERENCES platform.tenants(id),
    CONSTRAINT fk_security_policy_user FOREIGN KEY (updated_by) REFERENCES auth.users(id)
);

CREATE TABLE auth.approval_delegations (
    id                  INT             NOT NULL IDENTITY(1,1),
    tenant_id           INT             NOT NULL,
    delegator_user_id   INT             NOT NULL,
    delegate_user_id    INT             NOT NULL,
    start_date          DATE            NOT NULL,
    end_date            DATE            NOT NULL,
    notes               NVARCHAR(500)   NULL,
    created_at          DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    revoked_at          DATETIME2(3)    NULL,
    CONSTRAINT pk_approval_delegations PRIMARY KEY (id),
    CONSTRAINT fk_delegations_tenant FOREIGN KEY (tenant_id) REFERENCES platform.tenants(id),
    CONSTRAINT fk_delegations_delegator FOREIGN KEY (delegator_user_id) REFERENCES auth.users(id),
    CONSTRAINT fk_delegations_delegate FOREIGN KEY (delegate_user_id) REFERENCES auth.users(id),
    CONSTRAINT chk_delegation_dates CHECK (end_date >= start_date),
    CONSTRAINT chk_delegation_self CHECK (delegator_user_id <> delegate_user_id)
);

-- =============================================================================
-- 3. REF — REFERENCE DATA
-- =============================================================================

CREATE TABLE ref.departments (
    id              INT             NOT NULL IDENTITY(1,1),
    tenant_id       INT             NOT NULL,
    name            NVARCHAR(200)   NOT NULL,
    code            NVARCHAR(20)    NOT NULL,
    parent_dept_id  INT             NULL,               -- hierarchical departments
    dept_head_user_id INT           NULL,
    status          NVARCHAR(20)    NOT NULL DEFAULT 'active'
                                    CHECK (status IN ('active','inactive','archived')),
    created_at      DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    updated_at      DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_departments PRIMARY KEY (id),
    CONSTRAINT uq_departments_code UNIQUE (tenant_id, code),
    CONSTRAINT fk_departments_tenant FOREIGN KEY (tenant_id) REFERENCES platform.tenants(id),
    CONSTRAINT fk_departments_parent FOREIGN KEY (parent_dept_id) REFERENCES ref.departments(id),
    CONSTRAINT fk_departments_head FOREIGN KEY (dept_head_user_id) REFERENCES auth.users(id)
);

CREATE TABLE ref.cost_centres (
    id              INT             NOT NULL IDENTITY(1,1),
    tenant_id       INT             NOT NULL,
    code            NVARCHAR(30)    NOT NULL,
    name            NVARCHAR(200)   NOT NULL,
    department_id   INT             NOT NULL,
    status          NVARCHAR(20)    NOT NULL DEFAULT 'active'
                                    CHECK (status IN ('active','inactive','archived')),
    created_at      DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    updated_at      DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_cost_centres PRIMARY KEY (id),
    CONSTRAINT uq_cost_centres_code UNIQUE (tenant_id, code),
    CONSTRAINT fk_cost_centres_tenant FOREIGN KEY (tenant_id) REFERENCES platform.tenants(id),
    CONSTRAINT fk_cost_centres_dept FOREIGN KEY (department_id) REFERENCES ref.departments(id)
);

CREATE TABLE ref.locations (
    id              INT             NOT NULL IDENTITY(1,1),
    tenant_id       INT             NOT NULL,
    region          NVARCHAR(100)   NOT NULL,
    country         NVARCHAR(100)   NOT NULL,
    city            NVARCHAR(100)   NOT NULL,
    office_name     NVARCHAR(200)   NULL,
    status          NVARCHAR(20)    NOT NULL DEFAULT 'active'
                                    CHECK (status IN ('active','inactive','archived')),
    CONSTRAINT pk_locations PRIMARY KEY (id),
    CONSTRAINT fk_locations_tenant FOREIGN KEY (tenant_id) REFERENCES platform.tenants(id)
);

CREATE TABLE ref.currencies (
    id              INT             NOT NULL IDENTITY(1,1),
    code            NVARCHAR(3)     NOT NULL,
    name            NVARCHAR(80)    NOT NULL,
    symbol          NVARCHAR(5)     NOT NULL,
    CONSTRAINT pk_currencies PRIMARY KEY (id),
    CONSTRAINT uq_currencies_code UNIQUE (code)
);

CREATE TABLE ref.fx_rates (
    id                  INT             NOT NULL IDENTITY(1,1),
    tenant_id           INT             NOT NULL,
    from_currency_id    INT             NOT NULL,
    to_currency_id      INT             NOT NULL,
    rate                DECIMAL(18,6)   NOT NULL,
    effective_date      DATE            NOT NULL,
    -- Optionally snapshotted to a budget period
    budget_period_id    INT             NULL,
    is_snapshot         BIT             NOT NULL DEFAULT 0,
    created_by          INT             NOT NULL,
    created_at          DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_fx_rates PRIMARY KEY (id),
    CONSTRAINT fk_fx_rates_tenant FOREIGN KEY (tenant_id) REFERENCES platform.tenants(id),
    CONSTRAINT fk_fx_from_currency FOREIGN KEY (from_currency_id) REFERENCES ref.currencies(id),
    CONSTRAINT fk_fx_to_currency FOREIGN KEY (to_currency_id) REFERENCES ref.currencies(id),
    CONSTRAINT fk_fx_created_by FOREIGN KEY (created_by) REFERENCES auth.users(id),
    CONSTRAINT chk_fx_diff_currencies CHECK (from_currency_id <> to_currency_id)
);

CREATE TABLE ref.cost_categories (
    id              INT             NOT NULL IDENTITY(1,1),
    tenant_id       INT             NOT NULL,
    name            NVARCHAR(150)   NOT NULL,
    budget_type     NVARCHAR(10)    NOT NULL
                                    CHECK (budget_type IN ('capex','opex','both')),
    status          NVARCHAR(20)    NOT NULL DEFAULT 'active'
                                    CHECK (status IN ('active','inactive','archived')),
    created_at      DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_cost_categories PRIMARY KEY (id),
    CONSTRAINT fk_cost_categories_tenant FOREIGN KEY (tenant_id) REFERENCES platform.tenants(id)
);

CREATE TABLE ref.support_maintenance_types (
    id              INT             NOT NULL IDENTITY(1,1),
    tenant_id       INT             NOT NULL,
    name            NVARCHAR(150)   NOT NULL,           -- e.g. 'Software Maintenance', 'Managed Service'
    status          NVARCHAR(20)    NOT NULL DEFAULT 'active'
                                    CHECK (status IN ('active','inactive','archived')),
    created_at      DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_sm_types PRIMARY KEY (id),
    CONSTRAINT fk_sm_types_tenant FOREIGN KEY (tenant_id) REFERENCES platform.tenants(id)
);

CREATE TABLE ref.rejection_reasons (
    id              INT             NOT NULL IDENTITY(1,1),
    tenant_id       INT             NOT NULL,
    code            NVARCHAR(60)    NOT NULL,
    description     NVARCHAR(500)   NOT NULL,
    category        NVARCHAR(80)    NULL,               -- 'financial', 'documentation', 'policy', etc.
    is_active       BIT             NOT NULL DEFAULT 1,
    created_at      DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_rejection_reasons PRIMARY KEY (id),
    CONSTRAINT uq_rejection_reasons_code UNIQUE (tenant_id, code),
    CONSTRAINT fk_rejection_reasons_tenant FOREIGN KEY (tenant_id) REFERENCES platform.tenants(id)
);

CREATE TABLE ref.vendors (
    id                  INT             NOT NULL IDENTITY(1,1),
    tenant_id           INT             NOT NULL,
    name                NVARCHAR(300)   NOT NULL,
    vendor_type         NVARCHAR(40)    NOT NULL
                                        CHECK (vendor_type IN ('supplier','licensor','contractor','consultant','managed_service')),
    status              NVARCHAR(30)    NOT NULL DEFAULT 'approved'
                                        CHECK (status IN ('preferred','approved','under_review','suspended','archived')),
    primary_currency_id INT             NULL,
    payment_terms_days  SMALLINT        NOT NULL DEFAULT 30,
    vat_number          NVARCHAR(50)    NULL,
    notes               NVARCHAR(2000)  NULL,
    created_at          DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    updated_at          DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_vendors PRIMARY KEY (id),
    CONSTRAINT fk_vendors_tenant FOREIGN KEY (tenant_id) REFERENCES platform.tenants(id),
    CONSTRAINT fk_vendors_currency FOREIGN KEY (primary_currency_id) REFERENCES ref.currencies(id)
);

CREATE TABLE ref.vendor_category_tags (
    id              INT             NOT NULL IDENTITY(1,1),
    vendor_id       INT             NOT NULL,
    tag             NVARCHAR(80)    NOT NULL,
    CONSTRAINT pk_vendor_tags PRIMARY KEY (id),
    CONSTRAINT uq_vendor_tag UNIQUE (vendor_id, tag),
    CONSTRAINT fk_vendor_tags_vendor FOREIGN KEY (vendor_id) REFERENCES ref.vendors(id)
);

-- =============================================================================
-- 4. FINANCE — BUDGET PERIODS
-- =============================================================================

CREATE TABLE finance.budget_periods (
    id              INT             NOT NULL IDENTITY(1,1),
    tenant_id       INT             NOT NULL,
    name            NVARCHAR(100)   NOT NULL,           -- e.g. 'Q1 FY2026'
    fiscal_year     SMALLINT        NOT NULL,
    quarter         TINYINT         NULL                -- 1-4, NULL for annual periods
                                    CHECK (quarter IN (1,2,3,4,NULL)),
    start_date      DATE            NOT NULL,
    end_date        DATE            NOT NULL,
    status          NVARCHAR(20)    NOT NULL DEFAULT 'open'
                                    CHECK (status IN ('open','soft_locked','hard_locked','closed','archived')),
    created_by      INT             NOT NULL,
    created_at      DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    updated_at      DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_budget_periods PRIMARY KEY (id),
    CONSTRAINT chk_period_dates CHECK (end_date > start_date),
    CONSTRAINT fk_budget_periods_tenant FOREIGN KEY (tenant_id) REFERENCES platform.tenants(id),
    CONSTRAINT fk_budget_periods_created_by FOREIGN KEY (created_by) REFERENCES auth.users(id)
);

-- =============================================================================
-- 5. FINANCE — BUDGETS & BUDGET LINES
-- =============================================================================

CREATE TABLE finance.budgets (
    id              INT             NOT NULL IDENTITY(1,1),
    tenant_id       INT             NOT NULL,
    period_id       INT             NOT NULL,
    department_id   INT             NOT NULL,
    name            NVARCHAR(300)   NOT NULL,
    status          NVARCHAR(30)    NOT NULL DEFAULT 'draft'
                                    CHECK (status IN ('draft','pending','approved','rejected','returned','closed')),
    submitted_at    DATETIME2(3)    NULL,
    submitted_by    INT             NULL,
    created_by      INT             NOT NULL,
    created_at      DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    updated_at      DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_budgets PRIMARY KEY (id),
    CONSTRAINT fk_budgets_tenant FOREIGN KEY (tenant_id) REFERENCES platform.tenants(id),
    CONSTRAINT fk_budgets_period FOREIGN KEY (period_id) REFERENCES finance.budget_periods(id),
    CONSTRAINT fk_budgets_department FOREIGN KEY (department_id) REFERENCES ref.departments(id),
    CONSTRAINT fk_budgets_submitted_by FOREIGN KEY (submitted_by) REFERENCES auth.users(id),
    CONSTRAINT fk_budgets_created_by FOREIGN KEY (created_by) REFERENCES auth.users(id)
);

CREATE TABLE finance.budget_lines (
    id                  INT             NOT NULL IDENTITY(1,1),
    budget_id           INT             NOT NULL,
    description         NVARCHAR(500)   NOT NULL,
    budget_type         NVARCHAR(10)    NOT NULL
                                        CHECK (budget_type IN ('capex','opex')),
    cost_category_id    INT             NULL,
    -- Primary cost centre (may be overridden by splits)
    cost_centre_id      INT             NULL,
    location_id         INT             NULL,
    amount              DECIMAL(15,2)   NOT NULL,
    currency_id         INT             NOT NULL,
    vendor_id           INT             NULL,
    justification       NVARCHAR(MAX)   NULL,
    status              NVARCHAR(30)    NOT NULL DEFAULT 'draft'
                                        CHECK (status IN ('draft','pending','approved','rejected','returned')),
    -- Contract flag
    is_contract_linked  BIT             NOT NULL DEFAULT 0,
    -- Carry-over provenance
    source_line_id      INT             NULL,           -- points to original line if this is a carry-over
    is_projection       BIT             NOT NULL DEFAULT 0, -- system-generated recurring projection
    projection_source_agreement_id INT  NULL,
    -- Audit
    created_by          INT             NOT NULL,
    created_at          DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    updated_at          DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_budget_lines PRIMARY KEY (id),
    CONSTRAINT fk_budget_lines_budget FOREIGN KEY (budget_id) REFERENCES finance.budgets(id),
    CONSTRAINT fk_budget_lines_category FOREIGN KEY (cost_category_id) REFERENCES ref.cost_categories(id),
    CONSTRAINT fk_budget_lines_cost_centre FOREIGN KEY (cost_centre_id) REFERENCES ref.cost_centres(id),
    CONSTRAINT fk_budget_lines_location FOREIGN KEY (location_id) REFERENCES ref.locations(id),
    CONSTRAINT fk_budget_lines_currency FOREIGN KEY (currency_id) REFERENCES ref.currencies(id),
    CONSTRAINT fk_budget_lines_vendor FOREIGN KEY (vendor_id) REFERENCES ref.vendors(id),
    CONSTRAINT fk_budget_lines_source FOREIGN KEY (source_line_id) REFERENCES finance.budget_lines(id),
    CONSTRAINT fk_budget_lines_created_by FOREIGN KEY (created_by) REFERENCES auth.users(id)
);

CREATE TABLE finance.budget_line_cost_splits (
    id              INT             NOT NULL IDENTITY(1,1),
    budget_line_id  INT             NOT NULL,
    cost_centre_id  INT             NOT NULL,
    percentage      DECIMAL(5,2)    NOT NULL            -- e.g. 60.00
                                    CHECK (percentage > 0 AND percentage <= 100),
    amount          DECIMAL(15,2)   NOT NULL,
    created_at      DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_budget_line_splits PRIMARY KEY (id),
    CONSTRAINT fk_splits_budget_line FOREIGN KEY (budget_line_id) REFERENCES finance.budget_lines(id),
    CONSTRAINT fk_splits_cost_centre FOREIGN KEY (cost_centre_id) REFERENCES ref.cost_centres(id)
);

CREATE TABLE finance.budget_line_cost_changes (
    id                  INT             NOT NULL IDENTITY(1,1),
    budget_line_id      INT             NOT NULL,
    previous_amount     DECIMAL(15,2)   NOT NULL,
    new_amount          DECIMAL(15,2)   NOT NULL,
    change_amount       DECIMAL(15,2)   NOT NULL,
    change_pct          DECIMAL(8,4)    NOT NULL,
    justification       NVARCHAR(MAX)   NOT NULL,
    requires_reapproval BIT             NOT NULL DEFAULT 0,
    changed_by          INT             NOT NULL,
    changed_at          DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_budget_line_changes PRIMARY KEY (id),
    CONSTRAINT fk_line_changes_line FOREIGN KEY (budget_line_id) REFERENCES finance.budget_lines(id),
    CONSTRAINT fk_line_changes_user FOREIGN KEY (changed_by) REFERENCES auth.users(id)
);

-- =============================================================================
-- 6. FINANCE — DOCUMENTS
-- =============================================================================

CREATE TABLE finance.document_attachments (
    id              INT             NOT NULL IDENTITY(1,1),
    tenant_id       INT             NOT NULL,
    entity_type     NVARCHAR(60)    NOT NULL,           -- 'budget_line', 'intake_request', 'agreement', 'po'
    entity_id       INT             NOT NULL,
    filename        NVARCHAR(500)   NOT NULL,
    s3_key          NVARCHAR(1000)  NOT NULL,
    file_size_bytes INT             NOT NULL,
    mime_type       NVARCHAR(120)   NOT NULL,
    uploaded_by     INT             NOT NULL,
    uploaded_at     DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_document_attachments PRIMARY KEY (id),
    CONSTRAINT fk_docs_tenant FOREIGN KEY (tenant_id) REFERENCES platform.tenants(id),
    CONSTRAINT fk_docs_uploaded_by FOREIGN KEY (uploaded_by) REFERENCES auth.users(id)
);

-- =============================================================================
-- 7. FINANCE — AGREEMENTS (CONTRACT LIFECYCLE)
-- =============================================================================

CREATE TABLE finance.agreements (
    id                          INT             NOT NULL IDENTITY(1,1),
    tenant_id                   INT             NOT NULL,
    budget_line_id              INT             NOT NULL,
    agreement_name              NVARCHAR(150)   NOT NULL,
    support_maintenance_type_id INT             NOT NULL,
    term_type                   NVARCHAR(15)    NOT NULL
                                                CHECK (term_type IN ('single_month','multi_month')),
    term_months                 SMALLINT        NOT NULL CHECK (term_months >= 1),
    start_date                  DATE            NOT NULL,
    end_date                    DATE            NOT NULL,   -- auto-calculated: start + term_months - 1 day
    annual_cost                 DECIMAL(15,2)   NOT NULL,
    monthly_cost                DECIMAL(15,2)   NOT NULL,   -- = annual_cost / 12
    cost_increase_pct           DECIMAL(6,3)    NOT NULL DEFAULT 0.000,
    auto_generate_renewal       BIT             NOT NULL DEFAULT 0,
    vendor_id                   INT             NOT NULL,
    contract_s3_key             NVARCHAR(1000)  NULL,
    notes                       NVARCHAR(2000)  NULL,
    status                      NVARCHAR(20)    NOT NULL DEFAULT 'active'
                                                CHECK (status IN ('active','expiring_soon','expired','renewed','lapsing')),
    created_by                  INT             NOT NULL,
    created_at                  DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    updated_at                  DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_agreements PRIMARY KEY (id),
    CONSTRAINT chk_agreement_dates CHECK (end_date >= start_date),
    CONSTRAINT fk_agreements_tenant FOREIGN KEY (tenant_id) REFERENCES platform.tenants(id),
    CONSTRAINT fk_agreements_budget_line FOREIGN KEY (budget_line_id) REFERENCES finance.budget_lines(id),
    CONSTRAINT fk_agreements_sm_type FOREIGN KEY (support_maintenance_type_id) REFERENCES ref.support_maintenance_types(id),
    CONSTRAINT fk_agreements_vendor FOREIGN KEY (vendor_id) REFERENCES ref.vendors(id),
    CONSTRAINT fk_agreements_created_by FOREIGN KEY (created_by) REFERENCES auth.users(id)
);

CREATE TABLE finance.agreement_expiry_alerts (
    id              INT             NOT NULL IDENTITY(1,1),
    agreement_id    INT             NOT NULL,
    alert_type      NVARCHAR(20)    NOT NULL
                                    CHECK (alert_type IN ('90_day','30_day','7_day','escalation')),
    triggered_at    DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    sent_at         DATETIME2(3)    NULL,
    CONSTRAINT pk_agreement_alerts PRIMARY KEY (id),
    CONSTRAINT uq_agreement_alert UNIQUE (agreement_id, alert_type),
    CONSTRAINT fk_alerts_agreement FOREIGN KEY (agreement_id) REFERENCES finance.agreements(id)
);

CREATE TABLE finance.agreement_alert_acknowledgements (
    id                          INT             NOT NULL IDENTITY(1,1),
    alert_id                    INT             NOT NULL,
    user_id                     INT             NOT NULL,
    acknowledgement_response    NVARCHAR(50)    NOT NULL
                                                CHECK (acknowledgement_response IN (
                                                    'renewal_in_progress',
                                                    'renewal_approved_awaiting_po',
                                                    'not_renewing_will_lapse',
                                                    'transferred_to_new_agreement',
                                                    'deferred'
                                                )),
    defer_to_month              NVARCHAR(20)    NULL,      -- e.g. 'April 2026'
    replacement_agreement_id    INT             NULL,
    notes                       NVARCHAR(1000)  NULL,
    acknowledged_at             DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_alert_acknowledgements PRIMARY KEY (id),
    CONSTRAINT uq_alert_acknowledgement UNIQUE (alert_id, user_id),
    CONSTRAINT fk_ack_alert FOREIGN KEY (alert_id) REFERENCES finance.agreement_expiry_alerts(id),
    CONSTRAINT fk_ack_user FOREIGN KEY (user_id) REFERENCES auth.users(id)
);

-- =============================================================================
-- 8. FINANCE — INTAKE REQUESTS (Budget Requestor → Department Head)
-- =============================================================================

CREATE TABLE finance.budget_intake_requests (
    id                  INT             NOT NULL IDENTITY(1,1),
    tenant_id           INT             NOT NULL,
    requester_user_id   INT             NOT NULL,
    dept_head_user_id   INT             NOT NULL,
    title               NVARCHAR(300)   NOT NULL,
    category            NVARCHAR(100)   NULL,
    budget_type         NVARCHAR(10)    NOT NULL
                                        CHECK (budget_type IN ('capex','opex')),
    estimated_amount    DECIMAL(15,2)   NOT NULL,
    currency_id         INT             NOT NULL,
    vendor_id           INT             NULL,
    preferred_vendor_name NVARCHAR(300) NULL,
    required_by_date    DATE            NULL,
    cost_centre_id      INT             NULL,
    justification       NVARCHAR(MAX)   NOT NULL,
    status              NVARCHAR(30)    NOT NULL DEFAULT 'draft'
                                        CHECK (status IN ('draft','submitted','approved',
                                               'rejected','clarification_requested','included_in_budget')),
    dept_head_decision_at   DATETIME2(3) NULL,
    dept_head_note          NVARCHAR(2000) NULL,
    rejection_reason        NVARCHAR(500) NULL,
    -- If included: link to budget line created from this request
    resulting_budget_line_id INT         NULL,
    created_at          DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    updated_at          DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_intake_requests PRIMARY KEY (id),
    CONSTRAINT fk_intake_tenant FOREIGN KEY (tenant_id) REFERENCES platform.tenants(id),
    CONSTRAINT fk_intake_requester FOREIGN KEY (requester_user_id) REFERENCES auth.users(id),
    CONSTRAINT fk_intake_dept_head FOREIGN KEY (dept_head_user_id) REFERENCES auth.users(id),
    CONSTRAINT fk_intake_currency FOREIGN KEY (currency_id) REFERENCES ref.currencies(id),
    CONSTRAINT fk_intake_vendor FOREIGN KEY (vendor_id) REFERENCES ref.vendors(id),
    CONSTRAINT fk_intake_cost_centre FOREIGN KEY (cost_centre_id) REFERENCES ref.cost_centres(id),
    CONSTRAINT fk_intake_budget_line FOREIGN KEY (resulting_budget_line_id) REFERENCES finance.budget_lines(id)
);

-- =============================================================================
-- 9. WORKFLOW — APPROVAL WORKFLOW
-- =============================================================================

CREATE TABLE workflow.approval_workflow_rules (
    id                  INT             NOT NULL IDENTITY(1,1),
    tenant_id           INT             NOT NULL,
    -- NULL department_id = applies globally across all departments
    department_id       INT             NULL,
    budget_type         NVARCHAR(10)    NULL            -- NULL = all types
                                        CHECK (budget_type IN ('capex','opex',NULL)),
    min_amount          DECIMAL(15,2)   NULL,           -- NULL = no lower bound
    max_amount          DECIMAL(15,2)   NULL,           -- NULL = no upper bound
    level_number        TINYINT         NOT NULL,       -- 1, 2, 3 etc.
    approver_role_id    INT             NOT NULL,
    requires_tpi        BIT             NOT NULL DEFAULT 0,
    escalation_days     TINYINT         NOT NULL DEFAULT 5,
    is_active           BIT             NOT NULL DEFAULT 1,
    created_at          DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_approval_rules PRIMARY KEY (id),
    CONSTRAINT fk_approval_rules_tenant FOREIGN KEY (tenant_id) REFERENCES platform.tenants(id),
    CONSTRAINT fk_approval_rules_dept FOREIGN KEY (department_id) REFERENCES ref.departments(id),
    CONSTRAINT fk_approval_rules_role FOREIGN KEY (approver_role_id) REFERENCES auth.roles(id)
);

CREATE TABLE workflow.budget_approvals (
    id                  INT             NOT NULL IDENTITY(1,1),
    budget_id           INT             NOT NULL,
    level_number        TINYINT         NOT NULL,
    approver_user_id    INT             NOT NULL,
    -- If delegated
    original_approver_id INT            NULL,
    is_delegated        BIT             NOT NULL DEFAULT 0,
    status              NVARCHAR(20)    NOT NULL DEFAULT 'pending'
                                        CHECK (status IN ('pending','approved','rejected',
                                               'returned','delegated','escalated','superseded')),
    decision_at         DATETIME2(3)    NULL,
    rejection_reason_id INT             NULL,
    rejection_comment   NVARCHAR(2000)  NULL,
    return_comment      NVARCHAR(2000)  NULL,
    escalated_at        DATETIME2(3)    NULL,
    escalated_to_user_id INT            NULL,
    created_at          DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    updated_at          DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_budget_approvals PRIMARY KEY (id),
    CONSTRAINT fk_approvals_budget FOREIGN KEY (budget_id) REFERENCES finance.budgets(id),
    CONSTRAINT fk_approvals_approver FOREIGN KEY (approver_user_id) REFERENCES auth.users(id),
    CONSTRAINT fk_approvals_original FOREIGN KEY (original_approver_id) REFERENCES auth.users(id),
    CONSTRAINT fk_approvals_rejection FOREIGN KEY (rejection_reason_id) REFERENCES ref.rejection_reasons(id),
    CONSTRAINT fk_approvals_escalated_to FOREIGN KEY (escalated_to_user_id) REFERENCES auth.users(id)
);

CREATE TABLE workflow.approval_tpi_confirmations (
    id              INT             NOT NULL IDENTITY(1,1),
    approval_id     INT             NOT NULL,
    confirmer_user_id INT           NOT NULL,
    comment         NVARCHAR(2000)  NOT NULL,
    confirmed_at    DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_tpi_confirmations PRIMARY KEY (id),
    CONSTRAINT fk_tpi_approval FOREIGN KEY (approval_id) REFERENCES workflow.budget_approvals(id),
    CONSTRAINT fk_tpi_confirmer FOREIGN KEY (confirmer_user_id) REFERENCES auth.users(id)
);

-- =============================================================================
-- 10. FINANCE — PURCHASE ORDERS & GOODS RECEIPT
-- =============================================================================

CREATE TABLE finance.purchase_orders (
    id                      INT             NOT NULL IDENTITY(1,1),
    tenant_id               INT             NOT NULL,
    budget_line_id          INT             NOT NULL,
    po_number               NVARCHAR(50)    NOT NULL,   -- human-readable ref e.g. PO-2026-0142
    vendor_id               INT             NOT NULL,
    description             NVARCHAR(500)   NOT NULL,
    total_value             DECIMAL(15,2)   NOT NULL,
    currency_id             INT             NOT NULL,
    expected_delivery_date  DATE            NULL,
    status                  NVARCHAR(30)    NOT NULL DEFAULT 'open'
                                            CHECK (status IN ('open','partially_received',
                                                   'fully_received','closed','disputed')),
    created_by              INT             NOT NULL,
    created_at              DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    closed_at               DATETIME2(3)    NULL,
    closed_by               INT             NULL,
    closure_type            NVARCHAR(30)    NULL        -- 'normal','force_close','fully_received'
                                            CHECK (closure_type IN ('normal','force_close',
                                                   'fully_received',NULL)),
    closure_justification   NVARCHAR(2000)  NULL,
    updated_at              DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_purchase_orders PRIMARY KEY (id),
    CONSTRAINT uq_po_number UNIQUE (tenant_id, po_number),
    CONSTRAINT fk_po_tenant FOREIGN KEY (tenant_id) REFERENCES platform.tenants(id),
    CONSTRAINT fk_po_budget_line FOREIGN KEY (budget_line_id) REFERENCES finance.budget_lines(id),
    CONSTRAINT fk_po_vendor FOREIGN KEY (vendor_id) REFERENCES ref.vendors(id),
    CONSTRAINT fk_po_currency FOREIGN KEY (currency_id) REFERENCES ref.currencies(id),
    CONSTRAINT fk_po_created_by FOREIGN KEY (created_by) REFERENCES auth.users(id),
    CONSTRAINT fk_po_closed_by FOREIGN KEY (closed_by) REFERENCES auth.users(id)
);

CREATE TABLE finance.po_lines (
    id                      INT             NOT NULL IDENTITY(1,1),
    po_id                   INT             NOT NULL,
    description             NVARCHAR(300)   NOT NULL,
    quantity_ordered        DECIMAL(10,3)   NOT NULL,
    unit_price              DECIMAL(15,4)   NOT NULL,
    line_total              DECIMAL(15,2)   NOT NULL,
    currency_id             INT             NOT NULL,
    expected_delivery_date  DATE            NULL,
    status                  NVARCHAR(20)    NOT NULL DEFAULT 'open'
                                            CHECK (status IN ('open','partially_received',
                                                   'fully_received','disputed')),
    created_at              DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_po_lines PRIMARY KEY (id),
    CONSTRAINT fk_po_lines_po FOREIGN KEY (po_id) REFERENCES finance.purchase_orders(id),
    CONSTRAINT fk_po_lines_currency FOREIGN KEY (currency_id) REFERENCES ref.currencies(id)
);

CREATE TABLE finance.goods_receipts (
    id                  INT             NOT NULL IDENTITY(1,1),
    po_id               INT             NOT NULL,
    po_line_id          INT             NOT NULL,
    received_quantity   DECIMAL(10,3)   NOT NULL,
    received_value      DECIMAL(15,2)   NOT NULL,       -- received_quantity * unit_price
    received_date       DATE            NOT NULL,
    delivery_reference  NVARCHAR(200)   NULL,
    condition           NVARCHAR(20)    NOT NULL DEFAULT 'accepted'
                                        CHECK (condition IN ('accepted','disputed')),
    notes               NVARCHAR(1000)  NULL,
    received_by         INT             NOT NULL,
    created_at          DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_goods_receipts PRIMARY KEY (id),
    CONSTRAINT fk_gr_po FOREIGN KEY (po_id) REFERENCES finance.purchase_orders(id),
    CONSTRAINT fk_gr_po_line FOREIGN KEY (po_line_id) REFERENCES finance.po_lines(id),
    CONSTRAINT fk_gr_received_by FOREIGN KEY (received_by) REFERENCES auth.users(id)
);

CREATE TABLE finance.po_disputes (
    id                  INT             NOT NULL IDENTITY(1,1),
    po_line_id          INT             NOT NULL,
    raised_by           INT             NOT NULL,
    raised_at           DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    description         NVARCHAR(2000)  NOT NULL,
    resolved_at         DATETIME2(3)    NULL,
    resolved_by         INT             NULL,
    resolution          NVARCHAR(20)    NULL
                                        CHECK (resolution IN ('accepted','partial_accepted','rejected',NULL)),
    resolution_notes    NVARCHAR(2000)  NULL,
    CONSTRAINT pk_po_disputes PRIMARY KEY (id),
    CONSTRAINT fk_disputes_po_line FOREIGN KEY (po_line_id) REFERENCES finance.po_lines(id),
    CONSTRAINT fk_disputes_raised_by FOREIGN KEY (raised_by) REFERENCES auth.users(id),
    CONSTRAINT fk_disputes_resolved_by FOREIGN KEY (resolved_by) REFERENCES auth.users(id)
);

-- =============================================================================
-- 11. FINANCE — ACTUALS
-- =============================================================================

CREATE TABLE finance.actuals (
    id                  INT             NOT NULL IDENTITY(1,1),
    tenant_id           INT             NOT NULL,
    budget_line_id      INT             NOT NULL,
    amount              DECIMAL(15,2)   NOT NULL,
    currency_id         INT             NOT NULL,
    transaction_date    DATE            NOT NULL,
    description         NVARCHAR(500)   NOT NULL,
    vendor_id           INT             NULL,
    invoice_reference   NVARCHAR(200)   NULL,
    source              NVARCHAR(30)    NOT NULL
                                        CHECK (source IN ('manual','po_receipt',
                                               'bulk_import','system')),
    -- Links back to goods receipt if source = 'po_receipt'
    goods_receipt_id    INT             NULL,
    entered_by          INT             NOT NULL,
    entered_at          DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    -- Amendment tracking
    amended_by          INT             NULL,
    amended_at          DATETIME2(3)    NULL,
    amendment_justification NVARCHAR(MAX) NULL,
    -- Soft delete (only Finance Controller with period reopen)
    is_deleted          BIT             NOT NULL DEFAULT 0,
    deleted_by          INT             NULL,
    deleted_at          DATETIME2(3)    NULL,
    CONSTRAINT pk_actuals PRIMARY KEY (id),
    CONSTRAINT fk_actuals_tenant FOREIGN KEY (tenant_id) REFERENCES platform.tenants(id),
    CONSTRAINT fk_actuals_budget_line FOREIGN KEY (budget_line_id) REFERENCES finance.budget_lines(id),
    CONSTRAINT fk_actuals_currency FOREIGN KEY (currency_id) REFERENCES ref.currencies(id),
    CONSTRAINT fk_actuals_vendor FOREIGN KEY (vendor_id) REFERENCES ref.vendors(id),
    CONSTRAINT fk_actuals_goods_receipt FOREIGN KEY (goods_receipt_id) REFERENCES finance.goods_receipts(id),
    CONSTRAINT fk_actuals_entered_by FOREIGN KEY (entered_by) REFERENCES auth.users(id),
    CONSTRAINT fk_actuals_amended_by FOREIGN KEY (amended_by) REFERENCES auth.users(id)
);

-- =============================================================================
-- 12. FINANCE — FORECASTS
-- =============================================================================

CREATE TABLE finance.budget_forecasts (
    id                  INT             NOT NULL IDENTITY(1,1),
    budget_line_id      INT             NOT NULL,
    forecast_method     NVARCHAR(20)    NOT NULL
                                        CHECK (forecast_method IN ('linear','committed_actuals','manual')),
    forecast_amount     DECIMAL(15,2)   NOT NULL,
    forecast_date       DATE            NOT NULL,       -- date forecast was calculated
    justification       NVARCHAR(MAX)   NULL,           -- required for manual forecasts
    version_number      SMALLINT        NOT NULL DEFAULT 1,
    rag_status          NVARCHAR(10)    NOT NULL DEFAULT 'green'
                                        CHECK (rag_status IN ('green','amber','red')),
    created_by          INT             NOT NULL,
    created_at          DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_budget_forecasts PRIMARY KEY (id),
    CONSTRAINT fk_forecasts_budget_line FOREIGN KEY (budget_line_id) REFERENCES finance.budget_lines(id),
    CONSTRAINT fk_forecasts_created_by FOREIGN KEY (created_by) REFERENCES auth.users(id)
);

-- =============================================================================
-- 13. FINANCE — PERIOD CLOSE
-- =============================================================================

CREATE TABLE finance.period_close_sessions (
    id              INT             NOT NULL IDENTITY(1,1),
    period_id       INT             NOT NULL,
    initiated_by    INT             NOT NULL,
    current_step    TINYINT         NOT NULL DEFAULT 1
                                    CHECK (current_step BETWEEN 1 AND 7),
    status          NVARCHAR(20)    NOT NULL DEFAULT 'in_progress'
                                    CHECK (status IN ('in_progress','completed','reopened','abandoned')),
    started_at      DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    completed_at    DATETIME2(3)    NULL,
    CONSTRAINT pk_close_sessions PRIMARY KEY (id),
    CONSTRAINT fk_close_sessions_period FOREIGN KEY (period_id) REFERENCES finance.budget_periods(id),
    CONSTRAINT fk_close_sessions_user FOREIGN KEY (initiated_by) REFERENCES auth.users(id)
);

CREATE TABLE finance.period_close_step_log (
    id              INT             NOT NULL IDENTITY(1,1),
    session_id      INT             NOT NULL,
    step_number     TINYINT         NOT NULL CHECK (step_number BETWEEN 1 AND 7),
    step_name       NVARCHAR(100)   NOT NULL,
    completed_by    INT             NOT NULL,
    completed_at    DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    notes           NVARCHAR(2000)  NULL,
    CONSTRAINT pk_close_step_log PRIMARY KEY (id),
    CONSTRAINT fk_close_step_session FOREIGN KEY (session_id) REFERENCES finance.period_close_sessions(id),
    CONSTRAINT fk_close_step_user FOREIGN KEY (completed_by) REFERENCES auth.users(id)
);

CREATE TABLE finance.carry_over_rules (
    id                      INT             NOT NULL IDENTITY(1,1),
    period_close_session_id INT             NOT NULL,
    budget_line_id          INT             NOT NULL,
    carry_over_type         NVARCHAR(20)    NOT NULL
                                            CHECK (carry_over_type IN ('full_remaining',
                                                   'committed_only','specified_amount','do_not_carry')),
    specified_amount        DECIMAL(15,2)   NULL,       -- populated if type = 'specified_amount'
    -- Resulting carry-over line created in next period
    resulting_line_id       INT             NULL,
    created_by              INT             NOT NULL,
    created_at              DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_carry_over_rules PRIMARY KEY (id),
    CONSTRAINT uq_carry_over_rule UNIQUE (period_close_session_id, budget_line_id),
    CONSTRAINT fk_carry_over_session FOREIGN KEY (period_close_session_id) REFERENCES finance.period_close_sessions(id),
    CONSTRAINT fk_carry_over_line FOREIGN KEY (budget_line_id) REFERENCES finance.budget_lines(id),
    CONSTRAINT fk_carry_over_resulting FOREIGN KEY (resulting_line_id) REFERENCES finance.budget_lines(id),
    CONSTRAINT fk_carry_over_user FOREIGN KEY (created_by) REFERENCES auth.users(id)
);

CREATE TABLE finance.period_reopen_log (
    id                  INT             NOT NULL IDENTITY(1,1),
    period_id           INT             NOT NULL,
    reopened_by         INT             NOT NULL,
    second_approver_id  INT             NOT NULL,
    justification       NVARCHAR(MAX)   NOT NULL,
    reopened_at         DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    re_closed_at        DATETIME2(3)    NULL,
    CONSTRAINT pk_period_reopen_log PRIMARY KEY (id),
    CONSTRAINT fk_reopen_period FOREIGN KEY (period_id) REFERENCES finance.budget_periods(id),
    CONSTRAINT fk_reopen_user FOREIGN KEY (reopened_by) REFERENCES auth.users(id),
    CONSTRAINT fk_reopen_second FOREIGN KEY (second_approver_id) REFERENCES auth.users(id)
);

-- =============================================================================
-- 14. AUDIT — IMMUTABLE AUDIT LOG
-- =============================================================================

CREATE TABLE audit.audit_log (
    id              BIGINT          NOT NULL IDENTITY(1,1),
    tenant_id       INT             NOT NULL,
    user_id         INT             NULL,               -- NULL for system events
    session_id      INT             NULL,
    event_category  NVARCHAR(40)    NOT NULL,           -- 'financial','security','approval','admin','ai'
    event_type      NVARCHAR(100)   NOT NULL,           -- e.g. 'budget_approved', 'user_locked'
    entity_type     NVARCHAR(60)    NULL,
    entity_id       INT             NULL,
    description     NVARCHAR(1000)  NOT NULL,
    before_state    NVARCHAR(MAX)   NULL,               -- JSON snapshot before change
    after_state     NVARCHAR(MAX)   NULL,               -- JSON snapshot after change
    ip_address      NVARCHAR(45)    NULL,
    user_agent      NVARCHAR(500)   NULL,
    -- Cryptographic chain
    entry_hash      NVARCHAR(128)   NOT NULL,           -- SHA-512 hash of this entry's fields
    previous_hash   NVARCHAR(128)   NULL,               -- hash of previous entry (blockchain-style)
    created_at      DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_audit_log PRIMARY KEY (id),
    CONSTRAINT fk_audit_tenant FOREIGN KEY (tenant_id) REFERENCES platform.tenants(id)
    -- Note: NO FK to users/sessions intentionally — audit log must survive user deletion
);

-- =============================================================================
-- 15. PLATFORM — NOTIFICATIONS
-- =============================================================================

CREATE TABLE platform.notifications (
    id                          INT             NOT NULL IDENTITY(1,1),
    tenant_id                   INT             NOT NULL,
    user_id                     INT             NOT NULL,
    notification_type           NVARCHAR(80)    NOT NULL,
    title                       NVARCHAR(300)   NOT NULL,
    body                        NVARCHAR(2000)  NOT NULL,
    entity_type                 NVARCHAR(60)    NULL,
    entity_id                   INT             NULL,
    -- Standard notifications: dismissed by marking is_read = 1
    is_read                     BIT             NOT NULL DEFAULT 0,
    read_at                     DATETIME2(3)    NULL,
    -- Expiry alerts: persist until explicitly acknowledged
    requires_acknowledgement    BIT             NOT NULL DEFAULT 0,
    acknowledged_at             DATETIME2(3)    NULL,
    acknowledgement_response    NVARCHAR(80)    NULL,
    created_at                  DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_notifications PRIMARY KEY (id),
    CONSTRAINT fk_notifications_tenant FOREIGN KEY (tenant_id) REFERENCES platform.tenants(id),
    CONSTRAINT fk_notifications_user FOREIGN KEY (user_id) REFERENCES auth.users(id)
);

CREATE TABLE platform.notification_preferences (
    id                  INT             NOT NULL IDENTITY(1,1),
    user_id             INT             NOT NULL,
    notification_type   NVARCHAR(80)    NOT NULL,
    channel             NVARCHAR(20)    NOT NULL
                                        CHECK (channel IN ('in_app','email','teams','slack')),
    is_enabled          BIT             NOT NULL DEFAULT 1,
    CONSTRAINT pk_notification_prefs PRIMARY KEY (id),
    CONSTRAINT uq_notification_pref UNIQUE (user_id, notification_type, channel),
    CONSTRAINT fk_notif_prefs_user FOREIGN KEY (user_id) REFERENCES auth.users(id)
);

-- =============================================================================
-- 16. PLATFORM — AI JUSTIFICATION
-- =============================================================================

CREATE TABLE platform.ai_justification_requests (
    id              INT             NOT NULL IDENTITY(1,1),
    tenant_id       INT             NOT NULL,
    user_id         INT             NOT NULL,
    entity_type     NVARCHAR(60)    NOT NULL,           -- 'budget_line', 'intake_request'
    entity_id       INT             NOT NULL,
    input_text      NVARCHAR(MAX)   NULL,
    output_text     NVARCHAR(MAX)   NOT NULL,
    model_provider  NVARCHAR(20)    NOT NULL
                                    CHECK (model_provider IN ('anthropic','ollama')),
    model_name      NVARCHAR(100)   NOT NULL,
    accepted        BIT             NULL,               -- NULL = not yet decided, 0 = discarded, 1 = accepted
    created_at      DATETIME2(3)    NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_ai_requests PRIMARY KEY (id),
    CONSTRAINT fk_ai_tenant FOREIGN KEY (tenant_id) REFERENCES platform.tenants(id),
    CONSTRAINT fk_ai_user FOREIGN KEY (user_id) REFERENCES auth.users(id)
);

-- =============================================================================
-- 17. INDEXES
-- =============================================================================

-- Users
CREATE INDEX ix_users_tenant_email     ON auth.users (tenant_id, email);
CREATE INDEX ix_users_status           ON auth.users (tenant_id, status);
CREATE INDEX ix_users_scim             ON auth.users (scim_external_id) WHERE scim_external_id IS NOT NULL;

-- Sessions
CREATE INDEX ix_sessions_token         ON auth.sessions (token_hash);
CREATE INDEX ix_sessions_user          ON auth.sessions (user_id, revoked_at);

-- Budget lines
CREATE INDEX ix_budget_lines_budget    ON finance.budget_lines (budget_id);
CREATE INDEX ix_budget_lines_contract  ON finance.budget_lines (is_contract_linked) WHERE is_contract_linked = 1;

-- Agreements
CREATE INDEX ix_agreements_end_date    ON finance.agreements (end_date, status);
CREATE INDEX ix_agreements_tenant      ON finance.agreements (tenant_id, status);

-- Approvals
CREATE INDEX ix_approvals_budget       ON workflow.budget_approvals (budget_id, status);
CREATE INDEX ix_approvals_approver     ON workflow.budget_approvals (approver_user_id, status);

-- Purchase orders
CREATE INDEX ix_po_budget_line         ON finance.purchase_orders (budget_line_id);
CREATE INDEX ix_po_status              ON finance.purchase_orders (tenant_id, status);

-- Actuals
CREATE INDEX ix_actuals_budget_line    ON finance.actuals (budget_line_id);
CREATE INDEX ix_actuals_date           ON finance.actuals (transaction_date);

-- Audit log
CREATE INDEX ix_audit_tenant_date      ON audit.audit_log (tenant_id, created_at DESC);
CREATE INDEX ix_audit_entity           ON audit.audit_log (entity_type, entity_id);
CREATE INDEX ix_audit_user             ON audit.audit_log (user_id, created_at DESC);
CREATE INDEX ix_audit_category         ON audit.audit_log (event_category, created_at DESC);

-- Notifications
CREATE INDEX ix_notif_user_unread      ON platform.notifications (user_id, is_read, created_at DESC);
CREATE INDEX ix_notif_unacked          ON platform.notifications (user_id, requires_acknowledgement, acknowledged_at);

-- FX rates
CREATE INDEX ix_fx_rates_pair_date     ON ref.fx_rates (from_currency_id, to_currency_id, effective_date DESC);

-- Vendors
CREATE INDEX ix_vendors_tenant_status  ON ref.vendors (tenant_id, status);

-- Intake requests
CREATE INDEX ix_intake_requester       ON finance.budget_intake_requests (requester_user_id, status);
CREATE INDEX ix_intake_dept_head       ON finance.budget_intake_requests (dept_head_user_id, status);

GO
