package api

import "context"

type traceKey struct{}

func withTraceContext(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceKey{}, traceID)
}

func traceIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(traceKey{}).(string)
	return v
}
