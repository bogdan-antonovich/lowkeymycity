package validator

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

// seededCities is the canonical labels the fake cities table holds —
// exactly as stored, mixed case and all.
var seededCities = []string{"New York, NY", "Portland, OR"}

// citiesTable returns a SqlExecutor whose GetCities serves labels, but
// honors a canceled context first — faithful to the generated layer, where
// a dead context kills the query before any row arrives.
func citiesTable(ctrl *gomock.Controller, labels []string) *MockSqlExecutor {
	sql := NewMockSqlExecutor(ctrl)
	sql.EXPECT().GetCities(gomock.Any()).DoAndReturn(
		func(ctx context.Context) ([]string, error) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			return labels, nil
		}).AnyTimes()
	return sql
}

// TestCityValidator exercises the documented contract of the validator
// trio: a fresh validator knows nothing, Init loads the set (and only a
// successful Init changes anything), and lookups ignore case and
// surrounding whitespace while answers come back canonical.
func TestCityValidator(t *testing.T) {
	t.Run("before Init every input is unknown", func(t *testing.T) {
		v := NewCityValidator(zap.NewNop())

		// "until Init succeeds it knows no cities" — even a perfectly good
		// canonical label is rejected
		assert.False(t, v.IsValid("New York, NY"))
		id, ok := v.GetCityID("New York, NY")
		assert.False(t, ok)
		assert.Empty(t, id)
	})

	t.Run("Init with a nil executor panics", func(t *testing.T) {
		v := NewCityValidator(zap.NewNop())

		// "must be non-nil — nil panics"
		assert.Panics(t, func() { _ = v.Init(t.Context(), nil) })
	})

	t.Run("after Init any spelling maps to the canonical label", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		v := NewCityValidator(zap.NewNop())
		require.NoError(t, v.Init(t.Context(), citiesTable(ctrl, seededCities)))

		// "matching ignores case and surrounding whitespace"
		for _, spelling := range []string{
			"New York, NY",     // exact
			"new york, ny",     // lowercased
			"NEW YORK, NY",     // shouted
			"  new YORK, ny  ", // padded and mangled
			"\tNew York, NY\n", // exotic whitespace
		} {
			assert.True(t, v.IsValid(spelling), "spelling %q should be valid", spelling)
			id, ok := v.GetCityID(spelling)
			assert.True(t, ok)
			// "exactly as stored in the cities table"
			assert.Equal(t, "New York, NY", id, "spelling %q should map to the canonical label", spelling)
		}
	})

	t.Run("unknown cities stay unknown after Init", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		v := NewCityValidator(zap.NewNop())
		require.NoError(t, v.Init(t.Context(), citiesTable(ctrl, seededCities)))

		assert.False(t, v.IsValid("Atlantis, XX"))
		id, ok := v.GetCityID("Atlantis, XX")
		assert.False(t, ok)
		assert.Empty(t, id)
	})

	t.Run("a failed Init keeps the previously loaded set serving", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		v := NewCityValidator(zap.NewNop())
		require.NoError(t, v.Init(t.Context(), citiesTable(ctrl, seededCities)))

		dbDown := errors.New("db: connection refused")
		broken := NewMockSqlExecutor(ctrl)
		broken.EXPECT().GetCities(gomock.Any()).Return(nil, dbDown)

		err := v.Init(t.Context(), broken)

		require.ErrorIs(t, err, dbDown, "the database error must surface unchanged")
		// "whatever set was loaded before keeps serving"
		assert.True(t, v.IsValid("Portland, OR"), "the old set must survive a failed reload")
	})

	t.Run("a successful re-Init replaces the whole set", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		v := NewCityValidator(zap.NewNop())
		require.NoError(t, v.Init(t.Context(), citiesTable(ctrl, seededCities)))
		require.NoError(t, v.Init(t.Context(), citiesTable(ctrl, []string{"Asheville, NC"})))

		// "atomically replaces the whole set" — replaced, not merged
		assert.True(t, v.IsValid("Asheville, NC"), "the new set serves")
		assert.False(t, v.IsValid("Portland, OR"), "cities absent from the reload are gone")
	})

	t.Run("a canceled context aborts the load with the context's error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		v := NewCityValidator(zap.NewNop())
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		err := v.Init(ctx, citiesTable(ctrl, seededCities))

		require.ErrorIs(t, err, context.Canceled)
		assert.False(t, v.IsValid("New York, NY"), "an aborted load must not half-fill the set")
	})

	t.Run("lookups are safe for concurrent use", func(t *testing.T) {
		// "Safe for concurrent use" — a smoke test that only bites under
		// -race: hammer lookups while a re-Init swaps the set underneath
		ctrl := gomock.NewController(t)
		v := NewCityValidator(zap.NewNop())
		require.NoError(t, v.Init(t.Context(), citiesTable(ctrl, seededCities)))

		var wg sync.WaitGroup
		for range 8 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for range 100 {
					v.IsValid("portland, or")
					v.GetCityID("new york, ny")
				}
			}()
		}
		require.NoError(t, v.Init(t.Context(), citiesTable(ctrl, seededCities)))
		wg.Wait()
	})
}

// TestExport exercises the validator's file export: nothing to write
// before Init, and after one the file holds the labels exactly as the
// query ordered them — the ranking the frontend autocomplete relies on.
func TestExport(t *testing.T) {
	t.Run("before Init there is nothing to export", func(t *testing.T) {
		v := NewCityValidator(zap.NewNop())

		assert.Error(t, v.Export(filepath.Join(t.TempDir(), "cities.json")))
	})

	t.Run("after Init the file holds the labels in query order", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		v := NewCityValidator(zap.NewNop())
		require.NoError(t, v.Init(t.Context(), citiesTable(ctrl, seededCities)))

		path := filepath.Join(t.TempDir(), "cities.json")
		require.NoError(t, v.Export(path))

		data, err := os.ReadFile(path)
		require.NoError(t, err)
		var got []string
		require.NoError(t, json.Unmarshal(data, &got))
		// "in the query's population-descending order" — as given, untouched
		assert.Equal(t, seededCities, got)
	})

	t.Run("an unwritable path surfaces the error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		v := NewCityValidator(zap.NewNop())
		require.NoError(t, v.Init(t.Context(), citiesTable(ctrl, seededCities)))

		// a directory is not a writable file
		assert.Error(t, v.Export(t.TempDir()))
	})
}
