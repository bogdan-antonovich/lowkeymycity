package results

import (
	"testing"

	"lowkeymycity/internal/testdb"
	"lowkeymycity/pkg/types"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResultsServiceIntegration runs the results service against a real
// postgres with the live schema: a row seeded the way the quiz service
// stores verdicts must come back exactly, arrays and JSONB included, and
// a missing id must be the documented pgx.ErrNoRows.
func TestResultsServiceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test needs docker")
	}
	pool := testdb.New(t)
	svc := NewResultsService(New(pool))

	// seed one verdict by hand, exercising every column type the row maps:
	// TEXT[], JSONB and the plain scalars
	_, err := pool.Exec(t.Context(),
		`INSERT INTO results (id, combination, mode, city, score, title, summary, green_flags, red_flags, alternatives, closing)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		storedVerdict.ID, `{"seeded":"by-hand"}`, storedVerdict.Mode, storedVerdict.City, storedVerdict.Score,
		storedVerdict.Title, storedVerdict.Summary, storedVerdict.GreenFlags, storedVerdict.RedFlags,
		`[{"city":"Bend, OR","blurb":"the sunny-side alternative"}]`, storedVerdict.Closing)
	require.NoError(t, err)

	t.Run("an issued id returns the result exactly as saved", func(t *testing.T) {
		res, err := svc.GetResult(t.Context(), storedVerdict.ID)

		require.NoError(t, err)
		assert.Equal(t, storedVerdict, res, "every column survives the round trip")
	})

	t.Run("a never-issued id is pgx.ErrNoRows and a zero result", func(t *testing.T) {
		res, err := svc.GetResult(t.Context(), "res-ghost")

		require.ErrorIs(t, err, pgx.ErrNoRows)
		assert.Equal(t, types.QuizResult{}, res)
	})
}
