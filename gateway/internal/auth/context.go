package auth

import "context"

type ctxKey struct{}

func WithUser(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, ctxKey{}, username)
}

func UserFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKey{}).(string); ok {
		return v
	}
	return ""
}
