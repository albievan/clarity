package middleware

import "context"

func ctxWith(ctx context.Context, key, val any) context.Context {
	return context.WithValue(ctx, key, val)
}
