package results

import (
	"errors"
	"testing"

	"lowkeymycity/pkg/types"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// storedVerdict is one verdict as the quiz service once saved it — the
// results service must hand it back exactly like this, under its id.
var storedVerdict = types.QuizResult{
	ID:           "res-1",
	Mode:         "city",
	City:         "Portland, OR",
	Score:        86,
	Title:        "portland is lowkey your soulmate",
	Summary:      "quiet streets, strong coffee, cancelable plans.",
	GreenFlags:   []string{"you and portland want the same things"},
	RedFlags:     []string{"the grey is real"},
	Alternatives: []types.Alternative{{City: "Bend, OR", Blurb: "the sunny-side alternative"}},
	Closing:      "visit once in february before you commit.",
}

// storedRow is storedVerdict the way the results table returns it.
func storedRow(v types.QuizResult) GetStoredResultRow {
	return GetStoredResultRow{
		ID: v.ID, Mode: v.Mode, City: v.City, Score: int32(v.Score),
		Title: v.Title, Summary: v.Summary, GreenFlags: v.GreenFlags,
		RedFlags: v.RedFlags, Alternatives: v.Alternatives, Closing: v.Closing,
	}
}

// TestGetResult exercises the documented contract of GetResult: a thin
// read — the stored result exactly as saved, pgx.ErrNoRows when the id
// doesn't exist, and any other database error unchanged.
func TestGetResult(t *testing.T) {
	t.Run("an issued id returns the result exactly as saved", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		sql := NewMockSqlExecutor(ctrl)
		sql.EXPECT().GetStoredResult(gomock.Any(), "res-1").Return(storedRow(storedVerdict), nil)
		svc := NewResultsService(sql)

		res, err := svc.GetResult(t.Context(), "res-1")

		require.NoError(t, err)
		assert.Equal(t, storedVerdict, res, "every stored field comes back untouched, id included")
	})

	t.Run("a never-issued id is pgx.ErrNoRows and a zero result", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		sql := NewMockSqlExecutor(ctrl)
		// faithful to the generated layer: a :one query with no row is
		// pgx.ErrNoRows and a zero row
		sql.EXPECT().GetStoredResult(gomock.Any(), "res-ghost").Return(GetStoredResultRow{}, pgx.ErrNoRows)
		svc := NewResultsService(sql)

		res, err := svc.GetResult(t.Context(), "res-ghost")

		// the contract names the sentinel: callers are meant to be able to
		// tell "no such result" apart from "database broke"
		require.ErrorIs(t, err, pgx.ErrNoRows)
		assert.Equal(t, types.QuizResult{}, res, "the result is zero-valued")
	})

	t.Run("any other database error passes through unchanged", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		dbDown := errors.New("db: connection refused")
		sql := NewMockSqlExecutor(ctrl)
		sql.EXPECT().GetStoredResult(gomock.Any(), "res-1").Return(GetStoredResultRow{}, dbDown)
		svc := NewResultsService(sql)

		res, err := svc.GetResult(t.Context(), "res-1")

		require.ErrorIs(t, err, dbDown, "the database error must surface unchanged")
		assert.Equal(t, types.QuizResult{}, res)
	})
}
