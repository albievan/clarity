-- =============================================================================
--  CLARITY — Budget Management Platform
--  Database Schema — MariaDB 10.6+
--  All IDs: INT AUTO_INCREMENT
--  Charset: utf8mb4 / Collation: utf8mb4_unicode_ci
--  Engine: InnoDB
-- =============================================================================

SET NAMES utf8mb4;
SET time_zone = '+00:00';
SET foreign_key_checks = 0;

-- =============================================================================
-- 1. PLATFORM — TENANCY
-- =============================================================================

CREATE TABLE tenants (
    id                          INT             NOT NULL AUTO_INCREMENT,
    name                        VARCHAR(200)    NOT NULL,
    slug                        VARCHAR(100)    NOT NULL COMMENT 'URL-safe unique identifier',
    status                      VARCHAR(20)     NOT NULL DEFAULT 'active'
                                                CHECK (status IN ('active','suspended','decommissioned')),
    timezone                    VARCHAR(60)     NOT NULL DEFAULT 'UTC',
    base_currency               VARCHAR(3)      NOT NULL DEFAULT 'GBP',
    fiscal_year_start_month     TINYINT UNSIGNED NOT NULL DEFAULT 1
                                                CHECK (fiscal_year_start_month IN (1,4,7,10)),
    max_quote_uploads           TINYINT UNSIGNED NOT NULL DEFAULT 3,
    created_at                  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at                  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_tenants_slug (slug)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


CREATE TABLE feature_flags (
    id              INT             NOT NULL AUTO_INCREMENT,
    tenant_id       INT             NOT NULL,
    flag_key        VARCHAR(100)    NOT NULL,
    is_enabled      TINYINT(1)      NOT NULL DEFAULT 1,
    description     VARCHAR(500)    NULL,
    updated_by      INT             NULL,
    updated_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_feature_flags_tenant_key (tenant_id, flag_key),
    KEY ix_feature_flags_tenant (tenant_id),
    CONSTRAINT fk_feature_flags_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


CREATE TABLE rate_limit_config (
    id                  INT             NOT NULL AUTO_INCREMENT,
    tenant_id           INT             NOT NULL,
    endpoint_category   VARCHAR(80)     NOT NULL COMMENT 'e.g. auth, standard_read, bulk_import',
    per_user_per_minute INT             NOT NULL,
    per_ip_per_minute   INT             NOT NULL,
    burst_limit         INT             NOT NULL,
    updated_by          INT             NULL,
    updated_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_rate_limit_category (tenant_id, endpoint_category),
    CONSTRAINT fk_rate_limit_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 2. AUTH — IDENTITY & ACCESS
-- =============================================================================

CREATE TABLE roles (
    id              INT             NOT NULL AUTO_INCREMENT,
    name            VARCHAR(80)     NOT NULL,
    description     VARCHAR(500)    NULL,
    is_system_role  TINYINT(1)      NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    UNIQUE KEY uq_roles_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO roles (name, description, is_system_role) VALUES
('IT Administrator',    'Full platform administration',                           1),
('Finance Controller',  'Full finance access, period close, cross-dept reporting', 1),
('Budget Approver',     'Approve/reject budget submissions',                       1),
('Department Head',     'Review and action intake requests from team',             1),
('Budget Owner',        'Create and manage departmental budgets',                  1),
('Budget Requestor',    'Submit budget item requests',                             1),
('Read Only',           'Read-only access to permitted budget data',               1);


CREATE TABLE users (
    id                  INT             NOT NULL AUTO_INCREMENT,
    tenant_id           INT             NOT NULL,
    email               VARCHAR(320)    NOT NULL,
    display_name        VARCHAR(200)    NOT NULL,
    password_hash       VARCHAR(255)    NULL COMMENT 'NULL for SSO-only users',
    mfa_enabled         TINYINT(1)      NOT NULL DEFAULT 0,
    mfa_secret_enc      VARCHAR(500)    NULL     COMMENT 'Vault-encrypted TOTP secret',
    mfa_backup_codes    TEXT            NULL     COMMENT 'JSON array of hashed backup codes',
    status              VARCHAR(30)     NOT NULL DEFAULT 'active'
                                        CHECK (status IN ('active','locked','suspended','deprovisioned','pending_setup')),
    failed_login_count  TINYINT UNSIGNED NOT NULL DEFAULT 0,
    locked_until        DATETIME        NULL,
    password_changed_at DATETIME        NULL,
    require_pw_change   TINYINT(1)      NOT NULL DEFAULT 1,
    last_login_at       DATETIME        NULL,
    last_login_ip       VARCHAR(45)     NULL,
    sso_provider        VARCHAR(40)     NULL     CHECK (sso_provider IN ('entra_id','google','apple')),
    sso_subject         VARCHAR(500)    NULL,
    scim_external_id    VARCHAR(500)    NULL,
    created_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_users_email (tenant_id, email),
    KEY ix_users_status (tenant_id, status),
    KEY ix_users_scim (scim_external_id),
    CONSTRAINT fk_users_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


CREATE TABLE user_roles (
    id              INT             NOT NULL AUTO_INCREMENT,
    user_id         INT             NOT NULL,
    role_id         INT             NOT NULL,
    department_id   INT             NULL COMMENT 'NULL = global tenant scope',
    assigned_by     INT             NOT NULL,
    assigned_at     DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at      DATETIME        NULL,
    revoked_by      INT             NULL,
    PRIMARY KEY (id),
    KEY ix_user_roles_user (user_id),
    KEY ix_user_roles_role (role_id),
    CONSTRAINT fk_user_roles_user       FOREIGN KEY (user_id)    REFERENCES users(id),
    CONSTRAINT fk_user_roles_role       FOREIGN KEY (role_id)    REFERENCES roles(id),
    CONSTRAINT fk_user_roles_assigned   FOREIGN KEY (assigned_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


CREATE TABLE sessions (
    id              INT             NOT NULL AUTO_INCREMENT,
    user_id         INT             NOT NULL,
    tenant_id       INT             NOT NULL,
    token_hash      VARCHAR(255)    NOT NULL COMMENT 'SHA-256 hash of session token',
    ip_address      VARCHAR(45)     NOT NULL,
    user_agent      VARCHAR(500)    NULL,
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at      DATETIME        NOT NULL,
    revoked_at      DATETIME        NULL,
    revoke_reason   VARCHAR(100)    NULL COMMENT 'logout, timeout, admin, scim_deprovision',
    PRIMARY KEY (id),
    UNIQUE KEY uq_sessions_token (token_hash),
    KEY ix_sessions_user (user_id, revoked_at),
    CONSTRAINT fk_sessions_user   FOREIGN KEY (user_id)   REFERENCES users(id),
    CONSTRAINT fk_sessions_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


CREATE TABLE password_history (
    id              INT             NOT NULL AUTO_INCREMENT,
    user_id         INT             NOT NULL,
    password_hash   VARCHAR(255)    NOT NULL,
    changed_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY ix_pw_history_user (user_id),
    CONSTRAINT fk_pw_history_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


CREATE TABLE security_policy (
    id              INT             NOT NULL AUTO_INCREMENT,
    tenant_id       INT             NOT NULL,
    setting_key     VARCHAR(100)    NOT NULL,
    setting_value   VARCHAR(500)    NOT NULL,
    updated_by      INT             NOT NULL,
    updated_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_security_policy (tenant_id, setting_key),
    CONSTRAINT fk_security_policy_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    CONSTRAINT fk_security_policy_user   FOREIGN KEY (updated_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


CREATE TABLE approval_delegations (
    id                  INT             NOT NULL AUTO_INCREMENT,
    tenant_id           INT             NOT NULL,
    delegator_user_id   INT             NOT NULL,
    delegate_user_id    INT             NOT NULL,
    start_date          DATE            NOT NULL,
    end_date            DATE            NOT NULL,
    notes               VARCHAR(500)    NULL,
    created_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at          DATETIME        NULL,
    PRIMARY KEY (id),
    KEY ix_delegations_delegator (delegator_user_id),
    KEY ix_delegations_delegate (delegate_user_id),
    CONSTRAINT fk_delegations_tenant    FOREIGN KEY (tenant_id)          REFERENCES tenants(id),
    CONSTRAINT fk_delegations_delegator FOREIGN KEY (delegator_user_id)  REFERENCES users(id),
    CONSTRAINT fk_delegations_delegate  FOREIGN KEY (delegate_user_id)   REFERENCES users(id),
    CHECK (end_date >= start_date),
    CHECK (delegator_user_id <> delegate_user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 3. REFERENCE DATA
-- =============================================================================

CREATE TABLE departments (
    id                  INT             NOT NULL AUTO_INCREMENT,
    tenant_id           INT             NOT NULL,
    name                VARCHAR(200)    NOT NULL,
    code                VARCHAR(20)     NOT NULL,
    parent_dept_id      INT             NULL,
    dept_head_user_id   INT             NULL,
    status              VARCHAR(20)     NOT NULL DEFAULT 'active'
                                        CHECK (status IN ('active','inactive','archived')),
    created_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_departments_code (tenant_id, code),
    KEY ix_departments_tenant (tenant_id),
    CONSTRAINT fk_departments_tenant FOREIGN KEY (tenant_id)          REFERENCES tenants(id),
    CONSTRAINT fk_departments_parent FOREIGN KEY (parent_dept_id)     REFERENCES departments(id),
    CONSTRAINT fk_departments_head   FOREIGN KEY (dept_head_user_id)  REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


CREATE TABLE cost_centres (
    id              INT             NOT NULL AUTO_INCREMENT,
    tenant_id       INT             NOT NULL,
    code            VARCHAR(30)     NOT NULL,
    name            VARCHAR(200)    NOT NULL,
    department_id   INT             NOT NULL,
    status          VARCHAR(20)     NOT NULL DEFAULT 'active'
                                    CHECK (status IN ('active','inactive','archived')),
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_cost_centres_code (tenant_id, code),
    KEY ix_cost_centres_dept (department_id),
    CONSTRAINT fk_cost_centres_tenant FOREIGN KEY (tenant_id)     REFERENCES tenants(id),
    CONSTRAINT fk_cost_centres_dept   FOREIGN KEY (department_id) REFERENCES departments(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


CREATE TABLE locations (
    id              INT             NOT NULL AUTO_INCREMENT,
    tenant_id       INT             NOT NULL,
    region          VARCHAR(100)    NOT NULL,
    country         VARCHAR(100)    NOT NULL,
    city            VARCHAR(100)    NOT NULL,
    office_name     VARCHAR(200)    NULL,
    status          VARCHAR(20)     NOT NULL DEFAULT 'active'
                                    CHECK (status IN ('active','inactive','archived')),
    PRIMARY KEY (id),
    KEY ix_locations_tenant (tenant_id),
    CONSTRAINT fk_locations_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


CREATE TABLE currencies (
    id              INT             NOT NULL AUTO_INCREMENT,
    code            VARCHAR(3)      NOT NULL,
    name            VARCHAR(80)     NOT NULL,
    symbol          VARCHAR(5)      NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_currencies_code (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO currencies (code, name, symbol) VALUES
('GBP', 'British Pound Sterling', '£'),
('USD', 'US Dollar',              '$'),
('EUR', 'Euro',                   '€'),
('AUD', 'Australian Dollar',      'A$'),
('CAD', 'Canadian Dollar',        'C$'),
('JPY', 'Japanese Yen',           '¥'),
('SGD', 'Singapore Dollar',       'S$'),
('CHF', 'Swiss Franc',            'CHF');


CREATE TABLE fx_rates (
    id                  INT             NOT NULL AUTO_INCREMENT,
    tenant_id           INT             NOT NULL,
    from_currency_id    INT             NOT NULL,
    to_currency_id      INT             NOT NULL,
    rate                DECIMAL(18,6)   NOT NULL,
    effective_date      DATE            NOT NULL,
    budget_period_id    INT             NULL COMMENT 'Non-null when this is a period snapshot',
    is_snapshot         TINYINT(1)      NOT NULL DEFAULT 0,
    created_by          INT             NOT NULL,
    created_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY ix_fx_rates_pair_date (from_currency_id, to_currency_id, effective_date),
    CONSTRAINT fk_fx_rates_tenant       FOREIGN KEY (tenant_id)         REFERENCES tenants(id),
    CONSTRAINT fk_fx_from_currency      FOREIGN KEY (from_currency_id)  REFERENCES currencies(id),
    CONSTRAINT fk_fx_to_currency        FOREIGN KEY (to_currency_id)    REFERENCES currencies(id),
    CONSTRAINT fk_fx_created_by         FOREIGN KEY (created_by)        REFERENCES users(id),
    CHECK (from_currency_id <> to_currency_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


CREATE TABLE cost_categories (
    id              INT             NOT NULL AUTO_INCREMENT,
    tenant_id       INT             NOT NULL,
    name            VARCHAR(150)    NOT NULL,
    budget_type     VARCHAR(10)     NOT NULL CHECK (budget_type IN ('capex','opex','both')),
    status          VARCHAR(20)     NOT NULL DEFAULT 'active'
                                    CHECK (status IN ('active','inactive','archived')),
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY ix_cost_categories_tenant (tenant_id),
    CONSTRAINT fk_cost_categories_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


CREATE TABLE support_maintenance_types (
    id              INT             NOT NULL AUTO_INCREMENT,
    tenant_id       INT             NOT NULL,
    name            VARCHAR(150)    NOT NULL,
    status          VARCHAR(20)     NOT NULL DEFAULT 'active'
                                    CHECK (status IN ('active','inactive','archived')),
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY ix_sm_types_tenant (tenant_id),
    CONSTRAINT fk_sm_types_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO support_maintenance_types (tenant_id, name, status) VALUES
(1, 'Software Maintenance',        'active'),
(1, 'Hardware Support',            'active'),
(1, 'Managed Service',             'active'),
(1, 'Professional Services Retainer', 'active'),
(1, 'Licence Subscription',        'active'),
(1, 'Insurance',                   'active'),
(1, 'Lease',                       'active');


CREATE TABLE rejection_reasons (
    id              INT             NOT NULL AUTO_INCREMENT,
    tenant_id       INT             NOT NULL,
    code            VARCHAR(60)     NOT NULL,
    description     VARCHAR(500)    NOT NULL,
    category        VARCHAR(80)     NULL,
    is_active       TINYINT(1)      NOT NULL DEFAULT 1,
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_rejection_reasons_code (tenant_id, code),
    CONSTRAINT fk_rejection_reasons_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO rejection_reasons (tenant_id, code, description, category) VALUES
(1, 'INSUFFICIENT_JUSTIFICATION',  'Insufficient business justification provided',                 'documentation'),
(1, 'BUDGET_EXCEEDED',             'Requested amount exceeds available budget',                   'financial'),
(1, 'INCORRECT_CLASSIFICATION',    'Incorrect CapEx/OpEx classification',                         'financial'),
(1, 'MISSING_DOCUMENTATION',       'Supporting documentation not provided',                       'documentation'),
(1, 'DEFERRED_NEXT_PERIOD',        'Deferred to next budget period',                              'timing'),
(1, 'COMMERCIALLY_UNCOMPETITIVE',  'Pricing is not commercially competitive — obtain new quotes', 'commercial'),
(1, 'POLICY_NON_COMPLIANT',        'Does not comply with procurement or finance policy',          'policy'),
(1, 'DUPLICATE_REQUEST',           'Duplicate of an existing approved budget line',               'process'),
(1, 'INCORRECT_PERIOD',            'Submitted against incorrect budget period',                   'process'),
(1, 'OTHER',                       'Other reason — see comment',                                  'other');


CREATE TABLE vendors (
    id                  INT             NOT NULL AUTO_INCREMENT,
    tenant_id           INT             NOT NULL,
    name                VARCHAR(300)    NOT NULL,
    vendor_type         VARCHAR(40)     NOT NULL
                                        CHECK (vendor_type IN ('supplier','licensor','contractor','consultant','managed_service')),
    status              VARCHAR(30)     NOT NULL DEFAULT 'approved'
                                        CHECK (status IN ('preferred','approved','under_review','suspended','archived')),
    primary_currency_id INT             NULL,
    payment_terms_days  SMALLINT        NOT NULL DEFAULT 30,
    vat_number          VARCHAR(50)     NULL,
    notes               TEXT            NULL,
    created_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY ix_vendors_tenant_status (tenant_id, status),
    CONSTRAINT fk_vendors_tenant   FOREIGN KEY (tenant_id)           REFERENCES tenants(id),
    CONSTRAINT fk_vendors_currency FOREIGN KEY (primary_currency_id) REFERENCES currencies(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


CREATE TABLE vendor_category_tags (
    id          INT             NOT NULL AUTO_INCREMENT,
    vendor_id   INT             NOT NULL,
    tag         VARCHAR(80)     NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_vendor_tag (vendor_id, tag),
    CONSTRAINT fk_vendor_tags_vendor FOREIGN KEY (vendor_id) REFERENCES vendors(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 4. BUDGET PERIODS
-- =============================================================================

CREATE TABLE budget_periods (
    id              INT             NOT NULL AUTO_INCREMENT,
    tenant_id       INT             NOT NULL,
    name            VARCHAR(100)    NOT NULL,
    fiscal_year     SMALLINT        NOT NULL,
    quarter         TINYINT         NULL CHECK (quarter IN (1,2,3,4)),
    start_date      DATE            NOT NULL,
    end_date        DATE            NOT NULL,
    status          VARCHAR(20)     NOT NULL DEFAULT 'open'
                                    CHECK (status IN ('open','soft_locked','hard_locked','closed','archived')),
    created_by      INT             NOT NULL,
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY ix_budget_periods_tenant (tenant_id, fiscal_year),
    CONSTRAINT fk_budget_periods_tenant      FOREIGN KEY (tenant_id)  REFERENCES tenants(id),
    CONSTRAINT fk_budget_periods_created_by  FOREIGN KEY (created_by) REFERENCES users(id),
    CHECK (end_date > start_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Add FK from fx_rates back to budget_periods (deferred due to forward reference)
ALTER TABLE fx_rates
    ADD CONSTRAINT fk_fx_rates_period FOREIGN KEY (budget_period_id) REFERENCES budget_periods(id);

-- =============================================================================
-- 5. BUDGETS & BUDGET LINES
-- =============================================================================

CREATE TABLE budgets (
    id              INT             NOT NULL AUTO_INCREMENT,
    tenant_id       INT             NOT NULL,
    period_id       INT             NOT NULL,
    department_id   INT             NOT NULL,
    name            VARCHAR(300)    NOT NULL,
    status          VARCHAR(30)     NOT NULL DEFAULT 'draft'
                                    CHECK (status IN ('draft','pending','approved','rejected','returned','closed')),
    submitted_at    DATETIME        NULL,
    submitted_by    INT             NULL,
    created_by      INT             NOT NULL,
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY ix_budgets_tenant_period (tenant_id, period_id),
    KEY ix_budgets_department (department_id),
    KEY ix_budgets_status (tenant_id, status),
    CONSTRAINT fk_budgets_tenant       FOREIGN KEY (tenant_id)    REFERENCES tenants(id),
    CONSTRAINT fk_budgets_period       FOREIGN KEY (period_id)    REFERENCES budget_periods(id),
    CONSTRAINT fk_budgets_department   FOREIGN KEY (department_id) REFERENCES departments(id),
    CONSTRAINT fk_budgets_submitted_by FOREIGN KEY (submitted_by) REFERENCES users(id),
    CONSTRAINT fk_budgets_created_by   FOREIGN KEY (created_by)   REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


CREATE TABLE budget_lines (
    id                              INT             NOT NULL AUTO_INCREMENT,
    budget_id                       INT             NOT NULL,
    description                     VARCHAR(500)    NOT NULL,
    budget_type                     VARCHAR(10)     NOT NULL CHECK (budget_type IN ('capex','opex')),
    cost_category_id                INT             NULL,
    cost_centre_id                  INT             NULL,
    location_id                     INT             NULL,
    amount                          DECIMAL(15,2)   NOT NULL,
    currency_id                     INT             NOT NULL,
    vendor_id                       INT             NULL,
    justification                   TEXT            NULL,
    status                          VARCHAR(30)     NOT NULL DEFAULT 'draft'
                                                    CHECK (status IN ('draft','pending','approved','rejected','returned')),
    is_contract_linked              TINYINT(1)      NOT NULL DEFAULT 0,
    source_line_id                  INT             NULL COMMENT 'Populated for carry-over lines',
    is_projection                   TINYINT(1)      NOT NULL DEFAULT 0,
    projection_source_agreement_id  INT             NULL,
    created_by                      INT             NOT NULL,
    created_at                      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at                      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY ix_budget_lines_budget (budget_id),
    KEY ix_budget_lines_contract (is_contract_linked),
    KEY ix_budget_lines_status (status),
    CONSTRAINT fk_budget_lines_budget      FOREIGN KEY (budget_id)        REFERENCES budgets(id),
    CONSTRAINT fk_budget_lines_category    FOREIGN KEY (cost_category_id) REFERENCES cost_categories(id),
    CONSTRAINT fk_budget_lines_cc          FOREIGN KEY (cost_centre_id)   REFERENCES cost_centres(id),
    CONSTRAINT fk_budget_lines_location    FOREIGN KEY (location_id)      REFERENCES locations(id),
    CONSTRAINT fk_budget_lines_currency    FOREIGN KEY (currency_id)      REFERENCES currencies(id),
    CONSTRAINT fk_budget_lines_vendor      FOREIGN KEY (vendor_id)        REFERENCES vendors(id),
    CONSTRAINT fk_budget_lines_source      FOREIGN KEY (source_line_id)   REFERENCES budget_lines(id),
    CONSTRAINT fk_budget_lines_created_by  FOREIGN KEY (created_by)       REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


CREATE TABLE budget_line_cost_splits (
    id              INT             NOT NULL AUTO_INCREMENT,
    budget_line_id  INT             NOT NULL,
    cost_centre_id  INT             NOT NULL,
    percentage      DECIMAL(5,2)    NOT NULL CHECK (percentage > 0 AND percentage <= 100),
    amount          DECIMAL(15,2)   NOT NULL,
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY ix_splits_budget_line (budget_line_id),
    CONSTRAINT fk_splits_budget_line  FOREIGN KEY (budget_line_id) REFERENCES budget_lines(id),
    CONSTRAINT fk_splits_cost_centre  FOREIGN KEY (cost_centre_id) REFERENCES cost_centres(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


CREATE TABLE budget_line_cost_changes (
    id              INT             NOT NULL AUTO_INCREMENT,
    budget_line_id  INT             NOT NULL,
    previous_amount DECIMAL(15,2)   NOT NULL,
    new_amount      DECIMAL(15,2)   NOT NULL,
    change_amount   DECIMAL(15,2)   NOT NULL,
    change_pct      DECIMAL(8,4)    NOT NULL,
    justification   TEXT            NOT NULL,
    requires_reapproval TINYINT(1)  NOT NULL DEFAULT 0,
    changed_by      INT             NOT NULL,
    changed_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY ix_line_changes_line (budget_line_id),
    CONSTRAINT fk_line_changes_line FOREIGN KEY (budget_line_id) REFERENCES budget_lines(id),
    CONSTRAINT fk_line_changes_user FOREIGN KEY (changed_by)     REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 6. DOCUMENT ATTACHMENTS
-- =============================================================================

CREATE TABLE document_attachments (
    id              INT             NOT NULL AUTO_INCREMENT,
    tenant_id       INT             NOT NULL,
    entity_type     VARCHAR(60)     NOT NULL COMMENT 'budget_line, intake_request, agreement, po',
    entity_id       INT             NOT NULL,
    filename        VARCHAR(500)    NOT NULL,
    s3_key          VARCHAR(1000)   NOT NULL,
    file_size_bytes INT             NOT NULL,
    mime_type       VARCHAR(120)    NOT NULL,
    uploaded_by     INT             NOT NULL,
    uploaded_at     DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY ix_docs_entity (entity_type, entity_id),
    CONSTRAINT fk_docs_tenant      FOREIGN KEY (tenant_id)    REFERENCES tenants(id),
    CONSTRAINT fk_docs_uploaded_by FOREIGN KEY (uploaded_by)  REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 7. AGREEMENTS (CONTRACT LIFECYCLE)
-- =============================================================================

CREATE TABLE agreements (
    id                          INT             NOT NULL AUTO_INCREMENT,
    tenant_id                   INT             NOT NULL,
    budget_line_id              INT             NOT NULL,
    agreement_name              VARCHAR(150)    NOT NULL,
    support_maintenance_type_id INT             NOT NULL,
    term_type                   VARCHAR(15)     NOT NULL CHECK (term_type IN ('single_month','multi_month')),
    term_months                 SMALLINT        NOT NULL CHECK (term_months >= 1),
    start_date                  DATE            NOT NULL,
    end_date                    DATE            NOT NULL COMMENT 'Auto-calculated: start + term_months - 1 day',
    annual_cost                 DECIMAL(15,2)   NOT NULL,
    monthly_cost                DECIMAL(15,2)   NOT NULL COMMENT 'annual_cost / 12',
    cost_increase_pct           DECIMAL(6,3)    NOT NULL DEFAULT 0.000,
    auto_generate_renewal       TINYINT(1)      NOT NULL DEFAULT 0,
    vendor_id                   INT             NOT NULL,
    contract_s3_key             VARCHAR(1000)   NULL,
    notes                       TEXT            NULL,
    status                      VARCHAR(20)     NOT NULL DEFAULT 'active'
                                                CHECK (status IN ('active','expiring_soon','expired','renewed','lapsing')),
    created_by                  INT             NOT NULL,
    created_at                  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at                  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY ix_agreements_end_date (end_date, status),
    KEY ix_agreements_tenant (tenant_id, status),
    KEY ix_agreements_budget_line (budget_line_id),
    CONSTRAINT fk_agreements_tenant      FOREIGN KEY (tenant_id)                   REFERENCES tenants(id),
    CONSTRAINT fk_agreements_budget_line FOREIGN KEY (budget_line_id)              REFERENCES budget_lines(id),
    CONSTRAINT fk_agreements_sm_type     FOREIGN KEY (support_maintenance_type_id) REFERENCES support_maintenance_types(id),
    CONSTRAINT fk_agreements_vendor      FOREIGN KEY (vendor_id)                   REFERENCES vendors(id),
    CONSTRAINT fk_agreements_created_by  FOREIGN KEY (created_by)                  REFERENCES users(id),
    CHECK (end_date >= start_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Back-fill FK from budget_lines to agreements
ALTER TABLE budget_lines
    ADD CONSTRAINT fk_budget_lines_proj_agreement
    FOREIGN KEY (projection_source_agreement_id) REFERENCES agreements(id);


CREATE TABLE agreement_expiry_alerts (
    id              INT             NOT NULL AUTO_INCREMENT,
    agreement_id    INT             NOT NULL,
    alert_type      VARCHAR(20)     NOT NULL CHECK (alert_type IN ('90_day','30_day','7_day','escalation')),
    triggered_at    DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    sent_at         DATETIME        NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_agreement_alert (agreement_id, alert_type),
    CONSTRAINT fk_alerts_agreement FOREIGN KEY (agreement_id) REFERENCES agreements(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


CREATE TABLE agreement_alert_acknowledgements (
    id                          INT             NOT NULL AUTO_INCREMENT,
    alert_id                    INT             NOT NULL,
    user_id                     INT             NOT NULL,
    acknowledgement_response    VARCHAR(50)     NOT NULL
                                                CHECK (acknowledgement_response IN (
                                                    'renewal_in_progress',
                                                    'renewal_approved_awaiting_po',
                                                    'not_renewing_will_lapse',
                                                    'transferred_to_new_agreement',
                                                    'deferred'
                                                )),
    defer_to_month              VARCHAR(20)     NULL,
    replacement_agreement_id    INT             NULL,
    notes                       TEXT            NULL,
    acknowledged_at             DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_alert_acknowledgement (alert_id, user_id),
    CONSTRAINT fk_ack_alert FOREIGN KEY (alert_id) REFERENCES agreement_expiry_alerts(id),
    CONSTRAINT fk_ack_user  FOREIGN KEY (user_id)  REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 8. INTAKE REQUESTS
-- =============================================================================

CREATE TABLE budget_intake_requests (
    id                          INT             NOT NULL AUTO_INCREMENT,
    tenant_id                   INT             NOT NULL,
    requester_user_id           INT             NOT NULL,
    dept_head_user_id           INT             NOT NULL,
    title                       VARCHAR(300)    NOT NULL,
    category                    VARCHAR(100)    NULL,
    budget_type                 VARCHAR(10)     NOT NULL CHECK (budget_type IN ('capex','opex')),
    estimated_amount            DECIMAL(15,2)   NOT NULL,
    currency_id                 INT             NOT NULL,
    vendor_id                   INT             NULL,
    preferred_vendor_name       VARCHAR(300)    NULL,
    required_by_date            DATE            NULL,
    cost_centre_id              INT             NULL,
    justification               TEXT            NOT NULL,
    status                      VARCHAR(40)     NOT NULL DEFAULT 'draft'
                                                CHECK (status IN ('draft','submitted','approved','rejected',
                                                       'clarification_requested','included_in_budget')),
    dept_head_decision_at       DATETIME        NULL,
    dept_head_note              TEXT            NULL,
    rejection_reason            VARCHAR(500)    NULL,
    resulting_budget_line_id    INT             NULL,
    created_at                  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at                  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY ix_intake_requester (requester_user_id, status),
    KEY ix_intake_dept_head (dept_head_user_id, status),
    CONSTRAINT fk_intake_tenant      FOREIGN KEY (tenant_id)              REFERENCES tenants(id),
    CONSTRAINT fk_intake_requester   FOREIGN KEY (requester_user_id)      REFERENCES users(id),
    CONSTRAINT fk_intake_dept_head   FOREIGN KEY (dept_head_user_id)      REFERENCES users(id),
    CONSTRAINT fk_intake_currency    FOREIGN KEY (currency_id)            REFERENCES currencies(id),
    CONSTRAINT fk_intake_vendor      FOREIGN KEY (vendor_id)              REFERENCES vendors(id),
    CONSTRAINT fk_intake_cc          FOREIGN KEY (cost_centre_id)         REFERENCES cost_centres(id),
    CONSTRAINT fk_intake_budget_line FOREIGN KEY (resulting_budget_line_id) REFERENCES budget_lines(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 9. APPROVAL WORKFLOW
-- =============================================================================

CREATE TABLE approval_workflow_rules (
    id                  INT             NOT NULL AUTO_INCREMENT,
    tenant_id           INT             NOT NULL,
    department_id       INT             NULL COMMENT 'NULL = applies globally',
    budget_type         VARCHAR(10)     NULL CHECK (budget_type IN ('capex','opex')),
    min_amount          DECIMAL(15,2)   NULL,
    max_amount          DECIMAL(15,2)   NULL,
    level_number        TINYINT         NOT NULL,
    approver_role_id    INT             NOT NULL,
    requires_tpi        TINYINT(1)      NOT NULL DEFAULT 0,
    escalation_days     TINYINT         NOT NULL DEFAULT 5,
    is_active           TINYINT(1)      NOT NULL DEFAULT 1,
    created_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY ix_workflow_rules_tenant (tenant_id),
    CONSTRAINT fk_approval_rules_tenant FOREIGN KEY (tenant_id)        REFERENCES tenants(id),
    CONSTRAINT fk_approval_rules_dept   FOREIGN KEY (department_id)    REFERENCES departments(id),
    CONSTRAINT fk_approval_rules_role   FOREIGN KEY (approver_role_id) REFERENCES roles(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


CREATE TABLE budget_approvals (
    id                      INT             NOT NULL AUTO_INCREMENT,
    budget_id               INT             NOT NULL,
    level_number            TINYINT         NOT NULL,
    approver_user_id        INT             NOT NULL,
    original_approver_id    INT             NULL,
    is_delegated            TINYINT(1)      NOT NULL DEFAULT 0,
    status                  VARCHAR(20)     NOT NULL DEFAULT 'pending'
                                            CHECK (status IN ('pending','approved','rejected',
                                                   'returned','delegated','escalated','superseded')),
    decision_at             DATETIME        NULL,
    rejection_reason_id     INT             NULL,
    rejection_comment       TEXT            NULL,
    return_comment          TEXT            NULL,
    escalated_at            DATETIME        NULL,
    escalated_to_user_id    INT             NULL,
    created_at              DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at              DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY ix_approvals_budget (budget_id, status),
    KEY ix_approvals_approver (approver_user_id, status),
    CONSTRAINT fk_approvals_budget      FOREIGN KEY (budget_id)             REFERENCES budgets(id),
    CONSTRAINT fk_approvals_approver    FOREIGN KEY (approver_user_id)      REFERENCES users(id),
    CONSTRAINT fk_approvals_original    FOREIGN KEY (original_approver_id)  REFERENCES users(id),
    CONSTRAINT fk_approvals_rejection   FOREIGN KEY (rejection_reason_id)   REFERENCES rejection_reasons(id),
    CONSTRAINT fk_approvals_escalated   FOREIGN KEY (escalated_to_user_id)  REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


CREATE TABLE approval_tpi_confirmations (
    id                  INT             NOT NULL AUTO_INCREMENT,
    approval_id         INT             NOT NULL,
    confirmer_user_id   INT             NOT NULL,
    comment             TEXT            NOT NULL,
    confirmed_at        DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    CONSTRAINT fk_tpi_approval  FOREIGN KEY (approval_id)        REFERENCES budget_approvals(id),
    CONSTRAINT fk_tpi_confirmer FOREIGN KEY (confirmer_user_id)  REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 10. PURCHASE ORDERS & GOODS RECEIPT
-- =============================================================================

CREATE TABLE purchase_orders (
    id                      INT             NOT NULL AUTO_INCREMENT,
    tenant_id               INT             NOT NULL,
    budget_line_id          INT             NOT NULL,
    po_number               VARCHAR(50)     NOT NULL,
    vendor_id               INT             NOT NULL,
    description             VARCHAR(500)    NOT NULL,
    total_value             DECIMAL(15,2)   NOT NULL,
    currency_id             INT             NOT NULL,
    expected_delivery_date  DATE            NULL,
    status                  VARCHAR(30)     NOT NULL DEFAULT 'open'
                                            CHECK (status IN ('open','partially_received',
                                                   'fully_received','closed','disputed')),
    created_by              INT             NOT NULL,
    created_at              DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    closed_at               DATETIME        NULL,
    closed_by               INT             NULL,
    closure_type            VARCHAR(30)     NULL
                                            CHECK (closure_type IN ('normal','force_close','fully_received')),
    closure_justification   TEXT            NULL,
    updated_at              DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_po_number (tenant_id, po_number),
    KEY ix_po_budget_line (budget_line_id),
    KEY ix_po_status (tenant_id, status),
    CONSTRAINT fk_po_tenant      FOREIGN KEY (tenant_id)     REFERENCES tenants(id),
    CONSTRAINT fk_po_budget_line FOREIGN KEY (budget_line_id) REFERENCES budget_lines(id),
    CONSTRAINT fk_po_vendor      FOREIGN KEY (vendor_id)     REFERENCES vendors(id),
    CONSTRAINT fk_po_currency    FOREIGN KEY (currency_id)   REFERENCES currencies(id),
    CONSTRAINT fk_po_created_by  FOREIGN KEY (created_by)    REFERENCES users(id),
    CONSTRAINT fk_po_closed_by   FOREIGN KEY (closed_by)     REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


CREATE TABLE po_lines (
    id                      INT             NOT NULL AUTO_INCREMENT,
    po_id                   INT             NOT NULL,
    description             VARCHAR(300)    NOT NULL,
    quantity_ordered        DECIMAL(10,3)   NOT NULL,
    unit_price              DECIMAL(15,4)   NOT NULL,
    line_total              DECIMAL(15,2)   NOT NULL,
    currency_id             INT             NOT NULL,
    expected_delivery_date  DATE            NULL,
    status                  VARCHAR(20)     NOT NULL DEFAULT 'open'
                                            CHECK (status IN ('open','partially_received','fully_received','disputed')),
    created_at              DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY ix_po_lines_po (po_id),
    CONSTRAINT fk_po_lines_po       FOREIGN KEY (po_id)       REFERENCES purchase_orders(id),
    CONSTRAINT fk_po_lines_currency FOREIGN KEY (currency_id) REFERENCES currencies(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


CREATE TABLE goods_receipts (
    id                  INT             NOT NULL AUTO_INCREMENT,
    po_id               INT             NOT NULL,
    po_line_id          INT             NOT NULL,
    received_quantity   DECIMAL(10,3)   NOT NULL,
    received_value      DECIMAL(15,2)   NOT NULL,
    received_date       DATE            NOT NULL,
    delivery_reference  VARCHAR(200)    NULL,
    condition_status    VARCHAR(20)     NOT NULL DEFAULT 'accepted'
                                        CHECK (condition_status IN ('accepted','disputed')),
    notes               TEXT            NULL,
    received_by         INT             NOT NULL,
    created_at          DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY ix_gr_po (po_id),
    KEY ix_gr_po_line (po_line_id),
    CONSTRAINT fk_gr_po          FOREIGN KEY (po_id)      REFERENCES purchase_orders(id),
    CONSTRAINT fk_gr_po_line     FOREIGN KEY (po_line_id) REFERENCES po_lines(id),
    CONSTRAINT fk_gr_received_by FOREIGN KEY (received_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


CREATE TABLE po_disputes (
    id                  INT             NOT NULL AUTO_INCREMENT,
    po_line_id          INT             NOT NULL,
    raised_by           INT             NOT NULL,
    raised_at           DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    description         TEXT            NOT NULL,
    resolved_at         DATETIME        NULL,
    resolved_by         INT             NULL,
    resolution          VARCHAR(20)     NULL CHECK (resolution IN ('accepted','partial_accepted','rejected')),
    resolution_notes    TEXT            NULL,
    PRIMARY KEY (id),
    KEY ix_disputes_po_line (po_line_id),
    CONSTRAINT fk_disputes_po_line    FOREIGN KEY (po_line_id)  REFERENCES po_lines(id),
    CONSTRAINT fk_disputes_raised_by  FOREIGN KEY (raised_by)   REFERENCES users(id),
    CONSTRAINT fk_disputes_resolved   FOREIGN KEY (resolved_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 11. ACTUALS
-- =============================================================================

CREATE TABLE actuals (
    id                      INT             NOT NULL AUTO_INCREMENT,
    tenant_id               INT             NOT NULL,
    budget_line_id          INT             NOT NULL,
    amount                  DECIMAL(15,2)   NOT NULL,
    currency_id             INT             NOT NULL,
    transaction_date        DATE            NOT NULL,
    description             VARCHAR(500)    NOT NULL,
    vendor_id               INT             NULL,
    invoice_reference       VARCHAR(200)    NULL,
    source                  VARCHAR(30)     NOT NULL CHECK (source IN ('manual','po_receipt','bulk_import','system')),
    goods_receipt_id        INT             NULL,
    entered_by              INT             NOT NULL,
    entered_at              DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    amended_by              INT             NULL,
    amended_at              DATETIME        NULL,
    amendment_justification TEXT            NULL,
    is_deleted              TINYINT(1)      NOT NULL DEFAULT 0,
    deleted_by              INT             NULL,
    deleted_at              DATETIME        NULL,
    PRIMARY KEY (id),
    KEY ix_actuals_budget_line (budget_line_id),
    KEY ix_actuals_date (transaction_date),
    KEY ix_actuals_tenant (tenant_id),
    CONSTRAINT fk_actuals_tenant       FOREIGN KEY (tenant_id)       REFERENCES tenants(id),
    CONSTRAINT fk_actuals_budget_line  FOREIGN KEY (budget_line_id)  REFERENCES budget_lines(id),
    CONSTRAINT fk_actuals_currency     FOREIGN KEY (currency_id)     REFERENCES currencies(id),
    CONSTRAINT fk_actuals_vendor       FOREIGN KEY (vendor_id)       REFERENCES vendors(id),
    CONSTRAINT fk_actuals_gr           FOREIGN KEY (goods_receipt_id) REFERENCES goods_receipts(id),
    CONSTRAINT fk_actuals_entered_by   FOREIGN KEY (entered_by)      REFERENCES users(id),
    CONSTRAINT fk_actuals_amended_by   FOREIGN KEY (amended_by)      REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 12. FORECASTS
-- =============================================================================

CREATE TABLE budget_forecasts (
    id              INT             NOT NULL AUTO_INCREMENT,
    budget_line_id  INT             NOT NULL,
    forecast_method VARCHAR(20)     NOT NULL CHECK (forecast_method IN ('linear','committed_actuals','manual')),
    forecast_amount DECIMAL(15,2)   NOT NULL,
    forecast_date   DATE            NOT NULL,
    justification   TEXT            NULL,
    version_number  SMALLINT        NOT NULL DEFAULT 1,
    rag_status      VARCHAR(10)     NOT NULL DEFAULT 'green' CHECK (rag_status IN ('green','amber','red')),
    created_by      INT             NOT NULL,
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY ix_forecasts_line (budget_line_id),
    CONSTRAINT fk_forecasts_line       FOREIGN KEY (budget_line_id) REFERENCES budget_lines(id),
    CONSTRAINT fk_forecasts_created_by FOREIGN KEY (created_by)     REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 13. PERIOD CLOSE
-- =============================================================================

CREATE TABLE period_close_sessions (
    id              INT             NOT NULL AUTO_INCREMENT,
    period_id       INT             NOT NULL,
    initiated_by    INT             NOT NULL,
    current_step    TINYINT         NOT NULL DEFAULT 1 CHECK (current_step BETWEEN 1 AND 7),
    status          VARCHAR(20)     NOT NULL DEFAULT 'in_progress'
                                    CHECK (status IN ('in_progress','completed','reopened','abandoned')),
    started_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at    DATETIME        NULL,
    PRIMARY KEY (id),
    KEY ix_close_sessions_period (period_id),
    CONSTRAINT fk_close_sessions_period FOREIGN KEY (period_id)    REFERENCES budget_periods(id),
    CONSTRAINT fk_close_sessions_user   FOREIGN KEY (initiated_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


CREATE TABLE period_close_step_log (
    id              INT             NOT NULL AUTO_INCREMENT,
    session_id      INT             NOT NULL,
    step_number     TINYINT         NOT NULL CHECK (step_number BETWEEN 1 AND 7),
    step_name       VARCHAR(100)    NOT NULL,
    completed_by    INT             NOT NULL,
    completed_at    DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    notes           TEXT            NULL,
    PRIMARY KEY (id),
    KEY ix_close_step_session (session_id),
    CONSTRAINT fk_close_step_session FOREIGN KEY (session_id)   REFERENCES period_close_sessions(id),
    CONSTRAINT fk_close_step_user    FOREIGN KEY (completed_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


CREATE TABLE carry_over_rules (
    id                          INT             NOT NULL AUTO_INCREMENT,
    period_close_session_id     INT             NOT NULL,
    budget_line_id              INT             NOT NULL,
    carry_over_type             VARCHAR(20)     NOT NULL
                                                CHECK (carry_over_type IN ('full_remaining',
                                                       'committed_only','specified_amount','do_not_carry')),
    specified_amount            DECIMAL(15,2)   NULL,
    resulting_line_id           INT             NULL,
    created_by                  INT             NOT NULL,
    created_at                  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_carry_over_rule (period_close_session_id, budget_line_id),
    CONSTRAINT fk_carry_over_session   FOREIGN KEY (period_close_session_id) REFERENCES period_close_sessions(id),
    CONSTRAINT fk_carry_over_line      FOREIGN KEY (budget_line_id)          REFERENCES budget_lines(id),
    CONSTRAINT fk_carry_over_resulting FOREIGN KEY (resulting_line_id)       REFERENCES budget_lines(id),
    CONSTRAINT fk_carry_over_user      FOREIGN KEY (created_by)              REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


CREATE TABLE period_reopen_log (
    id                  INT             NOT NULL AUTO_INCREMENT,
    period_id           INT             NOT NULL,
    reopened_by         INT             NOT NULL,
    second_approver_id  INT             NOT NULL,
    justification       TEXT            NOT NULL,
    reopened_at         DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    re_closed_at        DATETIME        NULL,
    PRIMARY KEY (id),
    KEY ix_reopen_period (period_id),
    CONSTRAINT fk_reopen_period FOREIGN KEY (period_id)          REFERENCES budget_periods(id),
    CONSTRAINT fk_reopen_user   FOREIGN KEY (reopened_by)        REFERENCES users(id),
    CONSTRAINT fk_reopen_second FOREIGN KEY (second_approver_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 14. AUDIT LOG (IMMUTABLE)
-- =============================================================================

CREATE TABLE audit_log (
    id              BIGINT          NOT NULL AUTO_INCREMENT,
    tenant_id       INT             NOT NULL,
    user_id         INT             NULL COMMENT 'NULL for system events — NOT a FK, survives user deletion',
    session_id      INT             NULL,
    event_category  VARCHAR(40)     NOT NULL COMMENT 'financial, security, approval, admin, ai',
    event_type      VARCHAR(100)    NOT NULL,
    entity_type     VARCHAR(60)     NULL,
    entity_id       INT             NULL,
    description     VARCHAR(1000)   NOT NULL,
    before_state    JSON            NULL COMMENT 'Snapshot of record before change',
    after_state     JSON            NULL COMMENT 'Snapshot of record after change',
    ip_address      VARCHAR(45)     NULL,
    user_agent      VARCHAR(500)    NULL,
    entry_hash      VARCHAR(128)    NOT NULL COMMENT 'SHA-512 of this entry',
    previous_hash   VARCHAR(128)    NULL     COMMENT 'SHA-512 of previous entry (chain)',
    created_at      DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY ix_audit_tenant_date  (tenant_id, created_at),
    KEY ix_audit_entity       (entity_type, entity_id),
    KEY ix_audit_user         (user_id, created_at),
    KEY ix_audit_category     (event_category, created_at),
    CONSTRAINT fk_audit_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    -- Deliberately NO FK on user_id — audit log survives user deletion/deprovisioning
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
  COMMENT='Immutable audit log. Rows must never be updated or deleted.';

-- Protect audit_log from UPDATE and DELETE via stored procedure / application layer.
-- For additional protection, grant only INSERT+SELECT on this table to the app DB user.

-- =============================================================================
-- 15. NOTIFICATIONS
-- =============================================================================

CREATE TABLE notifications (
    id                          INT             NOT NULL AUTO_INCREMENT,
    tenant_id                   INT             NOT NULL,
    user_id                     INT             NOT NULL,
    notification_type           VARCHAR(80)     NOT NULL,
    title                       VARCHAR(300)    NOT NULL,
    body                        TEXT            NOT NULL,
    entity_type                 VARCHAR(60)     NULL,
    entity_id                   INT             NULL,
    is_read                     TINYINT(1)      NOT NULL DEFAULT 0,
    read_at                     DATETIME        NULL,
    requires_acknowledgement    TINYINT(1)      NOT NULL DEFAULT 0,
    acknowledged_at             DATETIME        NULL,
    acknowledgement_response    VARCHAR(80)     NULL,
    created_at                  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY ix_notif_user_unread (user_id, is_read, created_at),
    KEY ix_notif_unacked (user_id, requires_acknowledgement, acknowledged_at),
    CONSTRAINT fk_notifications_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    CONSTRAINT fk_notifications_user   FOREIGN KEY (user_id)   REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


CREATE TABLE notification_preferences (
    id                  INT             NOT NULL AUTO_INCREMENT,
    user_id             INT             NOT NULL,
    notification_type   VARCHAR(80)     NOT NULL,
    channel             VARCHAR(20)     NOT NULL CHECK (channel IN ('in_app','email','teams','slack')),
    is_enabled          TINYINT(1)      NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    UNIQUE KEY uq_notification_pref (user_id, notification_type, channel),
    CONSTRAINT fk_notif_prefs_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 16. AI JUSTIFICATION
-- =============================================================================

CREATE TABLE ai_justification_requests (
    id              INT             NOT NULL AUTO_INCREMENT,
    tenant_id       INT             NOT NULL,
    user_id         INT             NOT NULL,
    entity_type     VARCHAR(60)     NOT NULL COMMENT 'budget_line, intake_request',
    entity_id       INT             NOT NULL,
    input_text      TEXT            NULL,
    output_text     TEXT            NOT NULL,
    model_provider  VARCHAR(20)     NOT NULL CHECK (model_provider IN ('anthropic','ollama')),
    model_name      VARCHAR(100)    NOT NULL,
    accepted        TINYINT(1)      NULL COMMENT 'NULL = pending, 0 = discarded, 1 = accepted',
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY ix_ai_requests_entity (entity_type, entity_id),
    CONSTRAINT fk_ai_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    CONSTRAINT fk_ai_user   FOREIGN KEY (user_id)   REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =============================================================================
-- 17. USEFUL VIEWS
-- =============================================================================

-- Running balance view per budget line
CREATE VIEW vw_budget_line_running_balance AS
SELECT
    bl.id                                               AS budget_line_id,
    bl.budget_id,
    bl.description,
    bl.amount                                           AS total_budget,
    COALESCE(po_agg.committed, 0)                       AS committed,
    COALESCE(act_agg.actuals, 0)                        AS actuals_confirmed,
    bl.amount
        - COALESCE(po_agg.committed, 0)
        - COALESCE(act_agg.actuals, 0)                  AS remaining,
    ROUND(
        (COALESCE(po_agg.committed, 0) + COALESCE(act_agg.actuals, 0))
        / NULLIF(bl.amount, 0) * 100, 2
    )                                                   AS pct_committed_or_spent
FROM budget_lines bl
LEFT JOIN (
    SELECT budget_line_id, SUM(total_value) AS committed
    FROM purchase_orders
    WHERE status NOT IN ('closed')
    GROUP BY budget_line_id
) po_agg ON po_agg.budget_line_id = bl.id
LEFT JOIN (
    SELECT budget_line_id, SUM(amount) AS actuals
    FROM actuals
    WHERE is_deleted = 0
    GROUP BY budget_line_id
) act_agg ON act_agg.budget_line_id = bl.id;


-- Agreement expiry dashboard view
CREATE VIEW vw_agreement_expiry_dashboard AS
SELECT
    a.id,
    a.tenant_id,
    a.agreement_name,
    a.end_date,
    DATEDIFF(a.end_date, CURDATE())                     AS days_remaining,
    a.annual_cost,
    a.cost_increase_pct,
    ROUND(a.annual_cost * (1 + a.cost_increase_pct / 100), 2) AS projected_renewal_cost,
    v.name                                              AS vendor_name,
    smt.name                                            AS support_type,
    a.status,
    CASE
        WHEN DATEDIFF(a.end_date, CURDATE()) < 30  THEN 'red'
        WHEN DATEDIFF(a.end_date, CURDATE()) < 90  THEN 'amber'
        ELSE 'green'
    END                                                 AS rag_status,
    COUNT(ack.id)                                       AS acknowledgement_count
FROM agreements a
JOIN vendors v              ON v.id  = a.vendor_id
JOIN support_maintenance_types smt ON smt.id = a.support_maintenance_type_id
LEFT JOIN agreement_expiry_alerts al   ON al.agreement_id = a.id AND al.alert_type = '90_day'
LEFT JOIN agreement_alert_acknowledgements ack ON ack.alert_id = al.id
WHERE a.status IN ('active','expiring_soon')
GROUP BY a.id, a.tenant_id, a.agreement_name, a.end_date,
         a.annual_cost, a.cost_increase_pct, v.name, smt.name, a.status;


-- Department spend summary view
CREATE VIEW vw_department_spend_summary AS
SELECT
    b.department_id,
    d.name                              AS department_name,
    b.period_id,
    bp.name                             AS period_name,
    SUM(bl.amount)                      AS total_budget,
    COALESCE(SUM(po_agg.committed), 0)  AS total_committed,
    COALESCE(SUM(act_agg.actuals), 0)   AS total_actuals,
    SUM(bl.amount)
        - COALESCE(SUM(po_agg.committed), 0)
        - COALESCE(SUM(act_agg.actuals), 0) AS total_remaining
FROM budgets b
JOIN departments d       ON d.id  = b.department_id
JOIN budget_periods bp   ON bp.id = b.period_id
JOIN budget_lines bl     ON bl.budget_id = b.id
LEFT JOIN (
    SELECT budget_line_id, SUM(total_value) AS committed
    FROM purchase_orders WHERE status NOT IN ('closed')
    GROUP BY budget_line_id
) po_agg ON po_agg.budget_line_id = bl.id
LEFT JOIN (
    SELECT budget_line_id, SUM(amount) AS actuals
    FROM actuals WHERE is_deleted = 0
    GROUP BY budget_line_id
) act_agg ON act_agg.budget_line_id = bl.id
WHERE b.status = 'approved'
GROUP BY b.department_id, d.name, b.period_id, bp.name;

-- =============================================================================

SET foreign_key_checks = 1;
