package validator

import (
	"testing"

	"lowkeymycity/internal/testdb"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestCityValidatorIntegration loads the validator from a real postgres
// carrying the live cities seed — the same data production boots from —
// and checks the spelling-to-canonical-label mapping against it.
func TestCityValidatorIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test needs docker")
	}
	pool := testdb.New(t)
	v := NewCityValidator(zap.NewNop())
	require.NoError(t, v.Init(t.Context(), New(pool)), "Init loads the whole cities table")

	t.Run("seeded cities validate in any spelling", func(t *testing.T) {
		for _, spelling := range []string{"New York, NY", "  new york, ny ", "PORTLAND, OR"} {
			assert.True(t, v.IsValid(spelling), "%q names a seeded city", spelling)
		}
	})

	t.Run("the canonical label comes back exactly as stored", func(t *testing.T) {
		id, ok := v.GetCityID("  porTLand, or ")
		require.True(t, ok)
		assert.Equal(t, "Portland, OR", id)
	})

	t.Run("cities not in the table stay invalid", func(t *testing.T) {
		assert.False(t, v.IsValid("Atlantis, XX"))
		id, ok := v.GetCityID("Atlantis, XX")
		assert.False(t, ok)
		assert.Empty(t, id)
	})
}
