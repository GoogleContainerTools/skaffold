// Copyright 2026 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

import (
	"context"
	"log/slog"
)

type (
	ctxLogger struct{}
	ctxClient struct{}
)

// nopLogger is the logger returned by [LoggerFromContext] when the context
// carries none. It discards every record, so dispatch code can log
// unconditionally without a nil check.
var nopLogger = slog.New(slog.DiscardHandler)

// WithLogger returns a copy of ctx carrying logger.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxLogger{}, logger)
}

// LoggerFromContext extracts the [*slog.Logger] stored by [WithLogger]. When the
// context carries none it returns a process-wide nop logger that discards every
// record.
func LoggerFromContext(ctx context.Context) *slog.Logger {
	logger, ok := ctx.Value(ctxLogger{}).(*slog.Logger)
	if !ok {
		return nopLogger
	}

	return logger
}

// WithClient returns a copy of ctx carrying the [Client] dispatcher.
func WithClient(ctx context.Context, client Client) context.Context {
	return context.WithValue(ctx, ctxClient{}, client)
}

// ClientFromContext extracts the [Client] dispatcher stored by [WithClient]. The
// boolean reports whether a client was present.
func ClientFromContext(ctx context.Context) (Client, bool) {
	client, ok := ctx.Value(ctxClient{}).(Client)

	return client, ok
}
