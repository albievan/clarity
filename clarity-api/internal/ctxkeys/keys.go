package ctxkeys

type contextKey string

const (
	ClaimsKey    contextKey = "claims"
	TenantIDKey  contextKey = "tenant_id"
	RequestIDKey contextKey = "request_id"
)
