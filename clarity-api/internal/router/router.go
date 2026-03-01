package router

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/albievan/clarity/clarity-api/internal/config"
	"github.com/albievan/clarity/clarity-api/internal/db"
	"github.com/albievan/clarity/clarity-api/internal/middleware"

	domain_admin          "github.com/albievan/clarity/clarity-api/internal/domain/admin"
	domain_agreements     "github.com/albievan/clarity/clarity-api/internal/domain/agreements"
	domain_aijustify      "github.com/albievan/clarity/clarity-api/internal/domain/aijustification"
	domain_auditlog       "github.com/albievan/clarity/clarity-api/internal/domain/auditlog"
	domain_auth           "github.com/albievan/clarity/clarity-api/internal/domain/auth"
	domain_budgetlines    "github.com/albievan/clarity/clarity-api/internal/domain/budgetlines"
	domain_budgetperiods  "github.com/albievan/clarity/clarity-api/internal/domain/budgetperiods"
	domain_budgets        "github.com/albievan/clarity/clarity-api/internal/domain/budgets"
	domain_costcategories "github.com/albievan/clarity/clarity-api/internal/domain/costcategories"
	domain_costcentres    "github.com/albievan/clarity/clarity-api/internal/domain/costcentres"
	domain_currencies     "github.com/albievan/clarity/clarity-api/internal/domain/currencies"
	domain_delegations    "github.com/albievan/clarity/clarity-api/internal/domain/delegations"
	domain_departments    "github.com/albievan/clarity/clarity-api/internal/domain/departments"
	domain_documents      "github.com/albievan/clarity/clarity-api/internal/domain/documents"
	domain_forecasts      "github.com/albievan/clarity/clarity-api/internal/domain/forecasts"
	domain_fxrates        "github.com/albievan/clarity/clarity-api/internal/domain/fxrates"
	domain_intake         "github.com/albievan/clarity/clarity-api/internal/domain/intakerequests"
	domain_locations      "github.com/albievan/clarity/clarity-api/internal/domain/locations"
	domain_notifications  "github.com/albievan/clarity/clarity-api/internal/domain/notifications"
	domain_periodclose    "github.com/albievan/clarity/clarity-api/internal/domain/periodclose"
	domain_purchaseorders "github.com/albievan/clarity/clarity-api/internal/domain/purchaseorders"
	domain_actuals        "github.com/albievan/clarity/clarity-api/internal/domain/actuals"
	domain_rejections     "github.com/albievan/clarity/clarity-api/internal/domain/rejectionreasons"
	domain_smtypes        "github.com/albievan/clarity/clarity-api/internal/domain/smtypes"
	domain_users          "github.com/albievan/clarity/clarity-api/internal/domain/users"
	domain_vendors        "github.com/albievan/clarity/clarity-api/internal/domain/vendors"
	domain_workflow       "github.com/albievan/clarity/clarity-api/internal/domain/approvalworkflow"
)

// New builds and returns the fully mounted chi router.
func New(cfg *config.Config, database *db.DB, logger *slog.Logger) http.Handler {
	r := chi.NewRouter()

	// ── Global middleware
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.Logger)
	r.Use(chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://localhost:*", "http://127.0.0.1:*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "Idempotency-Key"},
		ExposedHeaders:   []string{"X-RateLimit-Limit", "X-RateLimit-Remaining", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	jwtSecret := cfg.JWT.Secret

	// ── Domain handler wiring
	usersSvc         := domain_users.NewService(domain_users.NewRepository(database))
	authHandler      := domain_auth.NewHandler(domain_auth.NewService(domain_auth.NewRepository(database), cfg.JWT), cfg.JWT)
	oauthHandler     := domain_auth.NewOAuthHandler(usersSvc, *cfg)
	usersHandler     := domain_users.NewHandler(usersSvc)
	delegHandler     := domain_delegations.NewHandler(domain_delegations.NewService(domain_delegations.NewRepository(database)))
	deptHandler      := domain_departments.NewHandler(domain_departments.NewService(domain_departments.NewRepository(database)))
	ccHandler        := domain_costcentres.NewHandler(domain_costcentres.NewService(domain_costcentres.NewRepository(database)))
	locHandler       := domain_locations.NewHandler(domain_locations.NewService(domain_locations.NewRepository(database)))
	curHandler       := domain_currencies.NewHandler(domain_currencies.NewService(domain_currencies.NewRepository(database)))
	fxHandler        := domain_fxrates.NewHandler(domain_fxrates.NewService(domain_fxrates.NewRepository(database)))
	catHandler       := domain_costcategories.NewHandler(domain_costcategories.NewService(domain_costcategories.NewRepository(database)))
	smHandler        := domain_smtypes.NewHandler(domain_smtypes.NewService(domain_smtypes.NewRepository(database)))
	rejHandler       := domain_rejections.NewHandler(domain_rejections.NewService(domain_rejections.NewRepository(database)))
	venHandler       := domain_vendors.NewHandler(domain_vendors.NewService(domain_vendors.NewRepository(database)))
	periodHandler    := domain_budgetperiods.NewHandler(domain_budgetperiods.NewService(domain_budgetperiods.NewRepository(database)))
	budgetHandler    := domain_budgets.NewHandler(domain_budgets.NewService(domain_budgets.NewRepository(database)))
	lineHandler      := domain_budgetlines.NewHandler(domain_budgetlines.NewService(domain_budgetlines.NewRepository(database)))
	agreementHandler := domain_agreements.NewHandler(domain_agreements.NewService(domain_agreements.NewRepository(database)))
	intakeHandler    := domain_intake.NewHandler(domain_intake.NewService(domain_intake.NewRepository(database)))
	workflowHandler  := domain_workflow.NewHandler(domain_workflow.NewService(domain_workflow.NewRepository(database)))
	poHandler        := domain_purchaseorders.NewHandler(domain_purchaseorders.NewService(domain_purchaseorders.NewRepository(database)))
	actualsHandler   := domain_actuals.NewHandler(domain_actuals.NewService(domain_actuals.NewRepository(database)))
	forecastHandler  := domain_forecasts.NewHandler(domain_forecasts.NewService(domain_forecasts.NewRepository(database)))
	closeHandler     := domain_periodclose.NewHandler(domain_periodclose.NewService(domain_periodclose.NewRepository(database)))
	auditHandler     := domain_auditlog.NewHandler(domain_auditlog.NewService(domain_auditlog.NewRepository(database)))
	notifHandler     := domain_notifications.NewHandler(domain_notifications.NewService(domain_notifications.NewRepository(database)))
	aiHandler        := domain_aijustify.NewHandler(domain_aijustify.NewService(domain_aijustify.NewRepository(database)))
	adminHandler     := domain_admin.NewHandler(domain_admin.NewService(domain_admin.NewRepository(database)))
	docHandler       := domain_documents.NewHandler(domain_documents.NewService(domain_documents.NewRepository(database)))

	r.Route("/v1", func(r chi.Router) {

		// ── Public endpoints (no JWT required)
		r.Post("/auth/login", authHandler.Login)
		r.Post("/auth/refresh", authHandler.Refresh)
		r.Post("/auth/password/reset-request", authHandler.PasswordResetRequest)
		r.Post("/auth/password/reset", authHandler.PasswordReset)
		r.Post("/auth/mfa/verify", authHandler.MFAVerify)
		r.Get("/currencies", curHandler.List)

		// ── OAuth 2.0 (Google + Apple) — public, browser-redirect flows
		r.Get("/auth/oauth/google/init", oauthHandler.GoogleInit)
		r.Get("/auth/oauth/google/callback", oauthHandler.GoogleCallback)
		r.Get("/auth/oauth/apple/init", oauthHandler.AppleInit)
		r.Post("/auth/oauth/apple/callback", oauthHandler.AppleCallback)

		// ── Authenticated routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(jwtSecret))

			// ── Auth (self-service)
			r.Post("/auth/logout", authHandler.Logout)
			r.Post("/auth/mfa/setup", authHandler.MFASetup)
			r.Post("/auth/mfa/confirm", authHandler.MFAConfirm)
			r.Post("/auth/mfa/disable", authHandler.MFADisable)
			r.Post("/auth/password/change", authHandler.PasswordChange)
			r.Get("/auth/sessions", authHandler.ListSessions)
			r.Delete("/auth/sessions/{sessionId}", authHandler.RevokeSession)

			// ── Users
			r.Get("/users", usersHandler.List)
			r.Post("/users", usersHandler.Create)
			r.Get("/users/{userId}", usersHandler.Get)
			r.Put("/users/{userId}", usersHandler.Update)
			r.Delete("/users/{userId}", usersHandler.Deprovision)
			r.Get("/users/{userId}/roles", usersHandler.ListRoles)
			r.Post("/users/{userId}/roles", usersHandler.AssignRole)
			r.Delete("/users/{userId}/roles/{assignmentId}", usersHandler.RevokeRole)
			r.Post("/users/{userId}/lock", usersHandler.Lock)
			r.Post("/users/{userId}/unlock", usersHandler.Unlock)
			r.Get("/users/{userId}/identities", usersHandler.ListIdentities)
			r.Delete("/users/{userId}/identities/{identityId}", usersHandler.DeleteIdentity)

			// ── Approval Delegations
			r.Get("/approval-delegations", delegHandler.List)
			r.Post("/approval-delegations", delegHandler.Create)
			r.Get("/approval-delegations/{delegationId}", delegHandler.Get)
			r.Post("/approval-delegations/{delegationId}/revoke", delegHandler.Revoke)

			// ── Departments
			r.Get("/departments", deptHandler.List)
			r.Post("/departments", deptHandler.Create)
			r.Get("/departments/{departmentId}", deptHandler.Get)
			r.Put("/departments/{departmentId}", deptHandler.Update)
			r.Delete("/departments/{departmentId}", deptHandler.Archive)

			// ── Cost Centres
			r.Get("/cost-centres", ccHandler.List)
			r.Post("/cost-centres", ccHandler.Create)
			r.Get("/cost-centres/{costCentreId}", ccHandler.Get)
			r.Put("/cost-centres/{costCentreId}", ccHandler.Update)

			// ── Locations
			r.Get("/locations", locHandler.List)
			r.Post("/locations", locHandler.Create)
			r.Put("/locations/{locationId}", locHandler.Update)

			// ── FX Rates
			r.Get("/fx-rates", fxHandler.List)
			r.Post("/fx-rates", fxHandler.Create)

			// ── Cost Categories
			r.Get("/cost-categories", catHandler.List)
			r.Post("/cost-categories", catHandler.Create)
			r.Put("/cost-categories/{categoryId}", catHandler.Update)

			// ── Support & Maintenance Types
			r.Get("/support-maintenance-types", smHandler.List)
			r.Post("/support-maintenance-types", smHandler.Create)

			// ── Rejection Reasons
			r.Get("/rejection-reasons", rejHandler.List)
			r.Post("/rejection-reasons", rejHandler.Create)

			// ── Vendors
			r.Get("/vendors", venHandler.List)
			r.Post("/vendors", venHandler.Create)
			r.Get("/vendors/{vendorId}", venHandler.Get)
			r.Put("/vendors/{vendorId}", venHandler.Update)

			// ── Budget Periods
			r.Get("/budget-periods", periodHandler.List)
			r.Post("/budget-periods", periodHandler.Create)
			r.Get("/budget-periods/{periodId}", periodHandler.Get)
			r.Put("/budget-periods/{periodId}", periodHandler.Update)
			r.Post("/budget-periods/{periodId}/reopen", periodHandler.Reopen)

			// ── Budgets
			r.Get("/budgets", budgetHandler.List)
			r.Post("/budgets", budgetHandler.Create)
			r.Get("/budgets/{budgetId}", budgetHandler.Get)
			r.Put("/budgets/{budgetId}", budgetHandler.Update)
			r.Delete("/budgets/{budgetId}", budgetHandler.Delete)
			r.Post("/budgets/{budgetId}/submit", budgetHandler.Submit)
			r.Post("/budgets/{budgetId}/approve", budgetHandler.Approve)
			r.Post("/budgets/{budgetId}/reject", budgetHandler.Reject)
			r.Post("/budgets/{budgetId}/return", budgetHandler.Return)
			r.Post("/budgets/{budgetId}/approvals/{approvalId}/tpi-confirm", budgetHandler.TPIConfirm)

			// ── Budget Lines
			r.Get("/budgets/{budgetId}/lines", lineHandler.List)
			r.Post("/budgets/{budgetId}/lines", lineHandler.Create)
			r.Get("/budget-lines/{lineId}", lineHandler.Get)
			r.Put("/budget-lines/{lineId}", lineHandler.Update)
			r.Delete("/budget-lines/{lineId}", lineHandler.Delete)
			r.Get("/budget-lines/{lineId}/documents", lineHandler.ListDocuments)
			r.Post("/budget-lines/{lineId}/documents", lineHandler.UploadDocument)
			r.Get("/budget-lines/{lineId}/forecasts", forecastHandler.List)
			r.Post("/budget-lines/{lineId}/forecasts", forecastHandler.Create)

			// ── Agreements
			r.Get("/agreements", agreementHandler.List)
			r.Post("/agreements", agreementHandler.Create)
			r.Get("/agreements/{agreementId}", agreementHandler.Get)
			r.Put("/agreements/{agreementId}", agreementHandler.Update)
			r.Post("/agreements/{agreementId}/cancel", agreementHandler.Cancel)
			r.Get("/agreements/{agreementId}/expiry-alerts", agreementHandler.ListAlerts)
			r.Post("/agreements/{agreementId}/expiry-alerts/{alertId}/acknowledge", agreementHandler.AcknowledgeAlert)

			// ── Intake Requests
			r.Get("/intake-requests", intakeHandler.List)
			r.Post("/intake-requests", intakeHandler.Create)
			r.Get("/intake-requests/{requestId}", intakeHandler.Get)
			r.Put("/intake-requests/{requestId}", intakeHandler.Update)
			r.Post("/intake-requests/{requestId}/submit", intakeHandler.Submit)
			r.Post("/intake-requests/{requestId}/approve", intakeHandler.Approve)
			r.Post("/intake-requests/{requestId}/reject", intakeHandler.Reject)
			r.Post("/intake-requests/{requestId}/convert", intakeHandler.Convert)

			// ── Approval Workflow Rules
			r.Get("/approval-workflow-rules", workflowHandler.List)
			r.Post("/approval-workflow-rules", workflowHandler.Create)
			r.Put("/approval-workflow-rules/{ruleId}", workflowHandler.Update)
			r.Delete("/approval-workflow-rules/{ruleId}", workflowHandler.Deactivate)
			r.Get("/approvals/pending", workflowHandler.PendingApprovals)

			// ── Purchase Orders
			r.Get("/purchase-orders", poHandler.List)
			r.Post("/purchase-orders", poHandler.Create)
			r.Get("/purchase-orders/{poId}", poHandler.Get)
			r.Put("/purchase-orders/{poId}", poHandler.Update)
			r.Post("/purchase-orders/{poId}/submit", poHandler.Submit)
			r.Post("/purchase-orders/{poId}/close", poHandler.Close)
			r.Get("/purchase-orders/{poId}/lines", poHandler.ListLines)
			r.Post("/purchase-orders/{poId}/lines", poHandler.AddLine)
			r.Put("/purchase-orders/{poId}/lines/{lineId}", poHandler.UpdateLine)
			r.Delete("/purchase-orders/{poId}/lines/{lineId}", poHandler.DeleteLine)
			r.Get("/purchase-orders/{poId}/receipts", poHandler.ListReceipts)
			r.Post("/purchase-orders/{poId}/receipts", poHandler.RecordReceipt)
			r.Get("/purchase-orders/{poId}/disputes", poHandler.ListDisputes)
			r.Post("/purchase-orders/{poId}/disputes", poHandler.RaiseDispute)
			r.Put("/purchase-orders/{poId}/disputes/{disputeId}/resolve", poHandler.ResolveDispute)

			// ── Actuals
			r.Get("/actuals", actualsHandler.List)
			r.Post("/actuals", actualsHandler.Create)
			r.Get("/actuals/{actualId}", actualsHandler.Get)
			r.Put("/actuals/{actualId}", actualsHandler.Amend)
			r.Post("/actuals/bulk-import", actualsHandler.BulkImport)
			r.Get("/actuals/bulk-import/{jobId}", actualsHandler.BulkImportStatus)

			// ── Period Close
			r.Get("/period-close/sessions", closeHandler.List)
			r.Post("/period-close/sessions", closeHandler.Create)
			r.Get("/period-close/sessions/{sessionId}", closeHandler.Get)
			r.Post("/period-close/sessions/{sessionId}/steps", closeHandler.CompleteStep)
			r.Get("/period-close/carry-over-rules", closeHandler.ListCarryOverRules)
			r.Post("/period-close/carry-over-rules", closeHandler.CreateCarryOverRule)

			// ── Audit Log
			r.Get("/audit-log", auditHandler.List)
			r.Get("/audit-log/{entryId}", auditHandler.Get)

			// ── Notifications
			r.Get("/notifications", notifHandler.List)
			r.Post("/notifications/{notificationId}/read", notifHandler.MarkRead)
			r.Post("/notifications/read-all", notifHandler.MarkAllRead)
			r.Get("/notifications/preferences", notifHandler.GetPreferences)
			r.Put("/notifications/preferences", notifHandler.UpdatePreferences)

			// ── AI Justification
			r.Post("/ai-justification/generate", aiHandler.Create)
			r.Get("/ai-justification/{requestId}", aiHandler.GetResult)

			// ── Admin
			r.Get("/admin/tenant", adminHandler.GetTenant)
			r.Put("/admin/tenant", adminHandler.UpdateTenant)
			r.Get("/admin/security-policy", adminHandler.GetSecurityPolicy)
			r.Put("/admin/security-policy", adminHandler.UpdateSecurityPolicy)
			r.Get("/admin/feature-flags", adminHandler.ListFeatureFlags)
			r.Put("/admin/feature-flags/{flagName}", adminHandler.UpdateFeatureFlag)

			// ── Documents
			r.Get("/documents/{documentId}", docHandler.GetDownloadURL)
			r.Delete("/documents/{documentId}", docHandler.Delete)
		})
	})

	// Health check — no auth, no rate limit.
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	_ = logger
	return r
}
