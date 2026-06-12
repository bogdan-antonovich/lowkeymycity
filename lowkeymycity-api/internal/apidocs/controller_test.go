package apidocs

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"lowkeymycity/docs"
	"lowkeymycity/internal/server"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newDocsAPI is a bare Echo with only the docs routes — the controller
// touches no service or middleware, so nothing else is wired.
func newDocsAPI() *echo.Echo {
	e := echo.New()
	NewDocsController().UI(e.Group(server.APIPrefix))
	return e
}

func do(e *echo.Echo, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestDocsUI(t *testing.T) {
	t.Run("index serves the UI pointed at our spec", func(t *testing.T) {
		for _, target := range []string{"/api/v1/docs", "/api/v1/docs/", "/api/v1/docs/index.html"} {
			rec := do(newDocsAPI(), httptest.NewRequest(http.MethodGet, target, nil))

			require.Equal(t, http.StatusOK, rec.Code, target)
			assert.Contains(t, rec.Header().Get(echo.HeaderContentType), "text/html", target)
			assert.Contains(t, rec.Body.String(), "SwaggerUIBundle", target)
			// our page, not the dist's petstore-pointing one
			assert.Contains(t, rec.Body.String(), "/api/v1/docs/swagger.json", target)
		}
	})

	t.Run("spec is the embedded swagger.json", func(t *testing.T) {
		rec := do(newDocsAPI(), httptest.NewRequest(http.MethodGet, "/api/v1/docs/swagger.json", nil))

		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get(echo.HeaderContentType), "application/json")
		assert.Equal(t, docs.Swagger, rec.Body.Bytes())
		// routes are relative in the spec; the prefix lives in basePath
		assert.Contains(t, rec.Body.String(), `"basePath": "`+server.APIPrefix+`"`)
		assert.Contains(t, rec.Body.String(), `"/quiz"`)
	})

	t.Run("static assets are served", func(t *testing.T) {
		for _, target := range []string{"/api/v1/docs/swagger-ui.css", "/api/v1/docs/swagger-ui-bundle.js"} {
			rec := do(newDocsAPI(), httptest.NewRequest(http.MethodGet, target, nil))

			require.Equal(t, http.StatusOK, rec.Code, target)
			assert.NotEmpty(t, rec.Body.Bytes(), target)
		}
	})

	t.Run("unknown asset is a 404", func(t *testing.T) {
		rec := do(newDocsAPI(), httptest.NewRequest(http.MethodGet, "/api/v1/docs/nope.js", nil))

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}
