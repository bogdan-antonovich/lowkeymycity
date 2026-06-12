package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// TestCarrier exercises the documented contract of Into/From/SetDefault: a
// logger put into a context comes back from it, and a context without one
// yields the package default — silent until SetDefault replaces it.
func TestCarrier(t *testing.T) {
	t.Run("From returns what Into stored", func(t *testing.T) {
		stored := zap.NewNop()

		got := From(Into(t.Context(), stored))

		assert.Equal(t, stored, got)
	})

	t.Run("From answers a bare context with the SetDefault logger", func(t *testing.T) {
		previous := def
		t.Cleanup(func() { SetDefault(previous) })

		boot := zap.NewNop()
		SetDefault(boot)

		assert.Equal(t, boot, From(t.Context()))
	})

	t.Run("From never returns nil, even before SetDefault", func(t *testing.T) {
		assert.NotNil(t, From(t.Context()))
	})
}

// TestWith exercises the documented contract of With: the returned logger
// prepends its fields to every line, on every level, without bleeding
// fields between calls.
func TestWith(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	log := With(zap.New(core), zap.String("request_id", "req-42"))

	log.Debug("entry", zap.String("city", "Portland, OR"))
	log.Info("milestone")
	log.Error("failure", zap.Error(assert.AnError))

	require.Equal(t, 3, logs.Len(), "every level passes through")
	for _, entry := range logs.All() {
		assert.Equal(t, "req-42", entry.ContextMap()["request_id"], "%q carries the shared field", entry.Message)
	}
	// per-call fields ride along exactly once and don't leak across calls
	assert.Equal(t, "Portland, OR", logs.All()[0].ContextMap()["city"])
	assert.NotContains(t, logs.All()[1].ContextMap(), "city")
}
