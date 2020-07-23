package log

import "context"

// logKey is a private context key.
type logKey struct{}

// NewContext returns a new context with logger.
func NewContext(ctx context.Context, v Interface) context.Context {
	return context.WithValue(ctx, logKey{}, v)
}

// FromContext returns the logger from context, or log.Log.
func FromContext(ctx context.Context) Interface {
	if v, ok := ctx.Value(logKey{}).(Interface); ok {
		return v
	}
	return Log
}
