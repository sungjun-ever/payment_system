package logger

import (
	"log/slog"
	"os"
)

func NewLogger() *slog.Logger {
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{})

	return slog.New(NewContextHandler(jsonHandler))
}
