package logger

import "context"

type contextKey string

const (
	requestIDKey contextKey = "request_id"
)

func (k contextKey) String() string {
	return string(k)
}

func WithRequestMeta(ctx context.Context, requestID string) context.Context {
	ctx = context.WithValue(ctx, requestIDKey, requestID)
	return ctx
}

func RequestIDFromContext(ctx context.Context) string {
	v, ok := ctx.Value(requestIDKey).(string)
	if !ok {
		return ""
	}
	return v
}
