package main

import (
	"context"
	"io"
	"log/slog"
)

type LoggingConfig struct {
	Output  io.Writer
	AppName string
	Debug   bool
}

func setupLogging(ctx context.Context, config *LoggingConfig) {
	logLevel := new(slog.LevelVar)
	logLevel.Set(slog.LevelError)
	if config.Debug {
		logLevel.Set(slog.LevelDebug)
	}
	opts := &slog.HandlerOptions{
		AddSource: config.Debug,
		Level:     logLevel,
	}

	handler := slog.NewJSONHandler(config.Output, opts)
	logger := slog.New(handler).With(
		"app", config.AppName,
	)

	slog.SetDefault(logger)
}
