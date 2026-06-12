// Package logging carries a request-scoped logger through context.Context
// so that every line written while serving one request shares its fields —
// at the edge that's the request id. Calls that arrive without one (boot
// paths, tests, direct use) get the package default, set once at startup.
package logging

import (
	"context"

	"lowkeymycity/pkg/types"

	"go.uber.org/zap"
)

type ctxKey struct{}

// def answers From when the context carries no logger. Until SetDefault
// runs it drops everything, so a process that skips SetDefault still
// works — it just stays silent outside requests.
var def types.Logger = zap.NewNop()

// SetDefault makes log the answer From gives for contexts that carry no
// logger. Call it once at startup before anything logs; it must not race
// with From.
func SetDefault(log types.Logger) {
	def = log
}

// Into returns a copy of ctx carrying log; From on the result returns log.
func Into(ctx context.Context, log types.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, log)
}

// From returns the logger carried by ctx, or the package default when ctx
// carries none.
func From(ctx context.Context) types.Logger {
	if log, ok := ctx.Value(ctxKey{}).(types.Logger); ok {
		return log
	}
	return def
}

// With returns a logger that prepends fields to every line written through
// it. Safe for concurrent use as long as base is.
func With(base types.Logger, fields ...zap.Field) types.Logger {
	// a real zap logger attaches fields natively, which keeps the caller
	// annotation on the call site; the generic wrapper below adds a frame
	// and would report itself as the caller of every line
	if z, ok := base.(*zap.Logger); ok {
		return z.With(fields...)
	}
	return &withFields{base: base, fields: fields}
}

type withFields struct {
	base   types.Logger
	fields []zap.Field
}

// merge builds a fresh slice every call — appending to w.fields directly
// could share its backing array between concurrent log calls.
func (w *withFields) merge(fields []zap.Field) []zap.Field {
	merged := make([]zap.Field, 0, len(w.fields)+len(fields))
	merged = append(merged, w.fields...)
	return append(merged, fields...)
}

func (w *withFields) Debug(msg string, fields ...zap.Field) { w.base.Debug(msg, w.merge(fields)...) }
func (w *withFields) Info(msg string, fields ...zap.Field)  { w.base.Info(msg, w.merge(fields)...) }
func (w *withFields) Error(msg string, fields ...zap.Field) { w.base.Error(msg, w.merge(fields)...) }
