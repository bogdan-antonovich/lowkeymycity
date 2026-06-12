package cities

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"lowkeymycity/internal/server"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newCitiesAPI is a bare Echo with only the cities route — the controller
// touches no service or middleware, so nothing else is wired.
func newCitiesAPI(path string) *echo.Echo {
	e := echo.New()
	NewCitiesController(path).GetCities(e.Group(server.APIPrefix))
	return e
}

func do(e *echo.Echo, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestGetCities(t *testing.T) {
	t.Run("serves the exported file with a day of cache", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "cities.json")
		require.NoError(t, os.WriteFile(path, []byte(`["New York, NY","Portland, OR"]`), 0o644))

		rec := do(newCitiesAPI(path), httptest.NewRequest(http.MethodGet, "/api/v1/cities", nil))

		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get(echo.HeaderContentType), "application/json")
		assert.Equal(t, "public, max-age=86400", rec.Header().Get("Cache-Control"))
		assert.JSONEq(t, `["New York, NY","Portland, OR"]`, rec.Body.String())
	})

	t.Run("a missing export is a 404", func(t *testing.T) {
		rec := do(newCitiesAPI(filepath.Join(t.TempDir(), "missing.json")),
			httptest.NewRequest(http.MethodGet, "/api/v1/cities", nil))

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}
