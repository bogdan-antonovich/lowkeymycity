package server

import (
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"lowkeymycity/pkg/logging"
	cityvalidator "lowkeymycity/pkg/validator"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// fakeCities is the minimal CityValidator the server tests need — these
// tests exercise the edge, not city resolution.
type fakeCities struct{}

func (fakeCities) Init(context.Context, cityvalidator.SqlExecutor) error { return nil }
func (fakeCities) Export(string) error                                   { return nil }
func (fakeCities) IsValid(city string) bool {
	return strings.EqualFold(strings.TrimSpace(city), "portland, or")
}
func (fakeCities) GetCityID(city string) (string, bool) {
	if (fakeCities{}).IsValid(city) {
		return "Portland, OR", true
	}
	return "", false
}

// newTestServer builds a fresh production server (fresh rate-limit buckets
// per test) with one harmless route outside the LLM set.
func newTestServer() *echo.Echo {
	e := New(zap.NewNop(), fakeCities{}, []string{"https://lowkeymycity.com"}, nil)
	e.GET("/other", func(c *echo.Context) error { return c.String(http.StatusOK, "ok") })
	return e
}

func do(e *echo.Echo, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// TestServerEdge exercises what the middleware chain promises: panics die
// as clean 500s, internals never leak to clients, the LLM endpoints sit
// behind the strict 15/min limit while everything else gets the generous
// one, CORS answers only the allowlist, and every response carries a
// request id.
func TestServerEdge(t *testing.T) {
	t.Run("a panic becomes a 500 that leaks nothing", func(t *testing.T) {
		e := newTestServer()
		e.GET("/boom", func(c *echo.Context) error { panic("secret panic detail") })

		rec := do(e, httptest.NewRequest(http.MethodGet, "/boom", nil))

		require.Equal(t, http.StatusInternalServerError, rec.Code, "the process survives and answers")
		assert.NotContains(t, rec.Body.String(), "secret", "panic internals stay out of the response")
	})

	t.Run("a handler error never leaks internals in the 500 body", func(t *testing.T) {
		e := newTestServer()
		e.GET("/fail", func(c *echo.Context) error { return errors.New("password=hunter2 dsn=postgres://") })

		rec := do(e, httptest.NewRequest(http.MethodGet, "/fail", nil))

		require.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.NotContains(t, rec.Body.String(), "hunter2", "error details are for logs, not clients")
	})

	t.Run("LLM endpoints allow 15 per minute, then deny", func(t *testing.T) {
		e := newTestServer()
		e.GET(APIPrefix+"/quiz", func(c *echo.Context) error { return c.String(http.StatusOK, "ok") })

		for i := 1; i <= 15; i++ {
			rec := do(e, httptest.NewRequest(http.MethodGet, "/api/v1/quiz?mode=match", nil))
			require.Equal(t, http.StatusOK, rec.Code, "request %d is within the budget", i)
		}
		rec := do(e, httptest.NewRequest(http.MethodGet, "/api/v1/quiz?mode=match", nil))
		assert.Equal(t, http.StatusTooManyRequests, rec.Code, "the 16th request in a minute is denied")
	})

	t.Run("other endpoints use the generous limiter", func(t *testing.T) {
		e := newTestServer()

		// 16 rapid requests would already be denied under the strict limit
		for i := 1; i <= 16; i++ {
			rec := do(e, httptest.NewRequest(http.MethodGet, "/other", nil))
			require.Equal(t, http.StatusOK, rec.Code, "request %d passes the usual limiter", i)
		}
	})

	t.Run("trusted proxy ranges are skipped when extracting the visitor ip", func(t *testing.T) {
		// the ip the rate limiter buckets by is whatever RealIP extracts,
		// so the assertion echoes it back. The chain simulates production:
		// visitor 203.0.113.7 → CDN edge 198.51.100.10 → nginx (private
		// socket peer), each proxy appending the hop it received.
		fromBehindCDN := func(e *echo.Echo) string {
			e.GET("/ip", func(c *echo.Context) error { return c.String(http.StatusOK, c.RealIP()) })
			req := httptest.NewRequest(http.MethodGet, "/ip", nil)
			req.RemoteAddr = "10.5.0.2:443"
			req.Header.Set(echo.HeaderXForwardedFor, "203.0.113.7, 198.51.100.10")
			return do(e, req).Body.String()
		}

		// without the trust option the edge is taken for the visitor — the
		// failure mode that pools every user into one rate-limit bucket
		e := New(zap.NewNop(), fakeCities{}, []string{"https://lowkeymycity.com"}, nil)
		assert.Equal(t, "198.51.100.10", fromBehindCDN(e), "an untrusted edge is taken for the visitor")

		// with the edge's range trusted the walk lands on the real visitor
		_, edgeRange, err := net.ParseCIDR("198.51.100.0/24")
		require.NoError(t, err)
		e = New(zap.NewNop(), fakeCities{}, []string{"https://lowkeymycity.com"}, []*net.IPNet{edgeRange})
		assert.Equal(t, "203.0.113.7", fromBehindCDN(e), "the visitor behind the trusted edge is extracted")
	})

	t.Run("CORS answers the allowlisted origin and ignores others", func(t *testing.T) {
		e := newTestServer()

		preflight := httptest.NewRequest(http.MethodOptions, "/other", nil)
		preflight.Header.Set(echo.HeaderOrigin, "https://lowkeymycity.com")
		preflight.Header.Set(echo.HeaderAccessControlRequestMethod, http.MethodGet)
		rec := do(e, preflight)
		assert.Equal(t, "https://lowkeymycity.com",
			rec.Header().Get(echo.HeaderAccessControlAllowOrigin), "the allowlisted origin is acknowledged")

		preflight = httptest.NewRequest(http.MethodOptions, "/other", nil)
		preflight.Header.Set(echo.HeaderOrigin, "https://evil.example")
		preflight.Header.Set(echo.HeaderAccessControlRequestMethod, http.MethodGet)
		rec = do(e, preflight)
		assert.Empty(t, rec.Header().Get(echo.HeaderAccessControlAllowOrigin), "unknown origins get nothing")
	})

	t.Run("every response carries a request id", func(t *testing.T) {
		e := newTestServer()

		rec := do(e, httptest.NewRequest(http.MethodGet, "/other", nil))

		assert.NotEmpty(t, rec.Header().Get(echo.HeaderXRequestID), "the id correlates access log and traces")
	})

	t.Run("logs written anywhere below a request group by its id", func(t *testing.T) {
		core, logs := observer.New(zap.DebugLevel)
		e := New(zap.New(core), fakeCities{}, []string{"https://lowkeymycity.com"}, nil)
		// a handler logging the way every service does: through the
		// context-carried logger, with its own fallback
		e.GET("/deep", func(c *echo.Context) error {
			logging.From(c.Request().Context()).Info("deep inside the request")
			return c.String(http.StatusOK, "ok")
		})

		rec := do(e, httptest.NewRequest(http.MethodGet, "/deep", nil))

		id := rec.Header().Get(echo.HeaderXRequestID)
		require.NotEmpty(t, id)
		deep := logs.FilterMessage("deep inside the request").All()
		require.Len(t, deep, 1)
		assert.Equal(t, id, deep[0].ContextMap()["request_id"],
			"the handler's line carries the same id the client saw")
		access := logs.FilterMessage("request").All()
		require.Len(t, access, 1)
		assert.Equal(t, id, access[0].ContextMap()["request_id"],
			"the access line and the deep line group together")
	})

	t.Run("oversized bodies are rejected", func(t *testing.T) {
		e := newTestServer()
		e.POST("/other", func(c *echo.Context) error { return c.String(http.StatusOK, "ok") })

		req := httptest.NewRequest(http.MethodPost, "/other", bytes.NewReader(make([]byte, maxBodyBytes+1)))
		rec := do(e, req)

		assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	})
}
