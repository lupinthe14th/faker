package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
)

type LoggingConfig struct {
	Output  io.Writer
	AppName string
	Debug   bool
}

func setupLogging(ctx context.Context, config *LoggingConfig) error {
	if config == nil {
		return errors.New("config is nil")
	}
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

	return nil
}
