# Clarity

**Clarity** is a secure, cloud-native **budget management platform**
designed to give organisations real-time visibility, structured
financial control, and full lifecycle governance over departmental
budgets.

It replaces fragmented spreadsheets and rigid legacy systems with a
modern, web-based solution that supports:

-   Multi-department budget management\
-   Real-time planned vs committed vs actual tracking\
-   Structured CapEx / OpEx governance\
-   Purchase order lifecycle management\
-   Vendor quote comparison and SOW control\
-   Multi-level approval workflows\
-   Forecasting, re-forecasting, and scenario planning\
-   Enterprise-grade security and compliance

------------------------------------------------------------------------

# 🚀 Core Capabilities

## 1. Flexible Budget Structure

Clarity supports a configurable, hierarchical budget model that can be
tailored per department.

Budgets can be structured by:

-   Budget Type: **CapEx, OpEx, Recurring OpEx**
-   Cost Category (e.g., labour, software, hardware, travel)
-   Department / Cost Centre
-   Project / Initiative
-   Vendor / Supplier
-   Geography / Region
-   Multi-currency with configurable base currency

Each department can define its own structure while maintaining
centralised reporting integrity.

------------------------------------------------------------------------

## 2. Real-Time Budget Tracking

Clarity provides a live financial ledger view for every budget line:

**Planned → Committed (POs) → Actuals → Remaining**

Key features:

-   Real-time running balance
-   Variance calculation (value and percentage)
-   Timestamped change history
-   Full user attribution
-   Immutable audit trail

------------------------------------------------------------------------

## 3. Purchase Order (PO) Management

Purchase Orders are first-class financial controls in Clarity.

-   POs link directly to budget line items
-   POs reduce available budget immediately (encumbrance model)
-   Multi-line PO support (qty × unit price calculations)
-   PO status tracking (open / partially received / closed)
-   PO-to-actual reconciliation
-   PO-to-budget traceability

------------------------------------------------------------------------

## 4. Vendor Quotes & SOW Management

Clarity includes a structured quote and Statement of Work module:

-   Upload multiple vendor quotes per budget item
-   Side-by-side quote comparison
-   Quote expiry tracking with automated alerts
-   Quote approval workflows
-   Preferred quote selection with mandatory justification
-   Direct import of approved quote line items into POs
-   SOW version history tracking

All vendor documents are securely stored in encrypted object storage
with controlled access.

------------------------------------------------------------------------

## 5. Configurable Approval Workflows

Clarity includes a rule-based, multi-level approval engine.

Approval rules can be configured based on:

-   Spend thresholds
-   Department
-   Budget type (CapEx / OpEx)
-   Request type (submission, amendment, reallocation)

Features include:

-   Up to 3 approval levels
-   Approve / Reject / Send Back with mandatory comments
-   Approval delegation
-   Full approval history log
-   Email notifications at every stage

------------------------------------------------------------------------

## 6. Forecasting & Scenario Planning

Clarity supports proactive financial management:

-   Quarterly re-forecasting cycles
-   Budget reallocation requests
-   Year-end carry-over management
-   What-If scenario planning sandbox
-   Shareable and promotable budget scenarios

------------------------------------------------------------------------

## 7. Dashboards & Reporting

Clarity provides executive and operational visibility through:

-   Executive Summary Dashboard (KPIs, variance, utilisation)
-   Department-level drilldowns
-   Project-level tracking
-   Vendor spend analysis
-   CapEx vs OpEx reporting
-   Consolidated cross-department views
-   Export to PDF and Excel

All dashboards support advanced filtering by fiscal year, department,
project, currency, and more.

------------------------------------------------------------------------

# 🔐 Security-First Architecture

Clarity is designed for sensitive financial and contractual data.
Security is enforced across every layer.

## Authentication & Access Control

-   OAuth 2.0 / OpenID Connect (OIDC)
-   Microsoft Entra ID (Azure AD) SSO
-   Google OIDC support
-   Apple Sign-In support
-   Local accounts with bcrypt hashing
-   TOTP Multi-Factor Authentication
-   Short-lived JWT tokens with refresh rotation
-   Role-Based Access Control (RBAC)

## Transport & Encryption

-   TLS 1.2+ enforced everywhere
-   Mutual TLS (mTLS) for service-to-service communication
-   Encrypted database connections
-   Encrypted object storage
-   Strict HTTP security headers (HSTS, CSP, X-Frame-Options, etc.)

## Secrets Management

-   Zero-local-configuration principle
-   All secrets stored in HashiCorp Vault or CyberArk
-   Vault Agent sidecar pattern
-   Automatic lease renewal
-   Dynamic secret rotation
-   Full audit logging of secret access

------------------------------------------------------------------------

# 🏗 Architecture Overview

Clarity is a cloud-native, containerised platform built on:

-   **Backend:** Go (Golang) microservices\
-   **Architecture Pattern:** Hexagonal Architecture (Ports & Adapters)\
-   **Frontend:** React + TypeScript SPA\
-   **API Contracts:** OpenAPI 3.0 (contract-first)\
-   **Database:** MariaDB (database-per-service pattern)\
-   **Object Storage:** S3-compatible (MinIO / AWS S3)\
-   **Secrets Management:** HashiCorp Vault or CyberArk\
-   **Deployment:** Docker + Docker Compose / Kubernetes

Each domain is implemented as an independent microservice.

------------------------------------------------------------------------

# 📦 Deployment Models

Clarity supports:

-   Cloud-hosted (SaaS) deployment
-   On-premise deployment
-   Docker Compose (single-host)
-   Kubernetes (production-grade scaling)

------------------------------------------------------------------------

# 📚 Documentation

Clarity ships with comprehensive documentation:

### End-User Documentation

-   Role-based User Manual (PDF & Word)
-   In-app contextual help
-   Interactive onboarding wizard
-   Knowledge base (Markdown, versioned)
-   Video walkthroughs
-   Structured release notes

### Administrator Documentation

-   Installation & Configuration Guide
-   Vault & Secrets Setup Guide
-   Infrastructure Runbook
-   Database & Migration Guide
-   Backup & Disaster Recovery Procedures
-   Security Hardening Checklist
-   Auto-generated Swagger API Reference

------------------------------------------------------------------------

# 📌 Vision

Clarity is built to become the authoritative source of truth for
organisational financial planning and control --- delivering:

-   Real-time transparency\
-   Structured governance\
-   Secure operations\
-   Scalable architecture\
-   Audit-ready compliance

It is not just a budgeting tool --- it is a financial governance
platform.

------------------------------------------------------------------------

# 📄 License

This project is confidential and proprietary.\
All rights reserved.
