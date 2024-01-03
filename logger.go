package main

import (
	"context"
	"io"
	"log/slog"
)

func setupLogging(ctx context.Context, output io.Writer, appName string, debug *bool) {
	logLevel := new(slog.LevelVar)
	logLevel.Set(slog.LevelError)
	if *debug {
		logLevel.Set(slog.LevelDebug)
	}
	opts := &slog.HandlerOptions{
		AddSource: *debug,
		Level:     logLevel,
	}

	handler := slog.NewJSONHandler(output, opts)
	logger := slog.New(handler).With(
		"app", app.Name,
	)

	slog.SetDefault(logger)
}
