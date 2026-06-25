// SPDX-FileCopyrightText: 2020 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

import (
	"context"

	"go.uber.org/zap"
)

var (
	ctxLogger struct{}
	ctxClient struct{}
)

// WithLogger returns the context with zap.Logger value.
func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, ctxLogger, logger)
}

// LoggerFromContext extracts zap.Logger from context.
func LoggerFromContext(ctx context.Context) *zap.Logger {
	logger, ok := ctx.Value(ctxLogger).(*zap.Logger)
	if !ok {
		return zap.NewNop()
	}

	return logger
}

// WithClient returns the context with Client value.
func WithClient(ctx context.Context, client Client) context.Context {
	return context.WithValue(ctx, ctxClient, client)
}
