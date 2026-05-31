package logger

import (
	"context"
	"log/slog"
)

type ContextHandler struct {
	slog.Handler
}

func NewContextHandler(h slog.Handler) *ContextHandler {
	return &ContextHandler{h}
}

func (h ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if requestID := RequestIDFromContext(ctx); requestID != "" {
		r.AddAttrs(slog.String(requestIDKey.String(), requestID))
	}

	return h.Handler.Handle(ctx, r)
}
