package health

import (
	"context"
	"net/http"
	"time"

	"lowkeymycity/pkg/logging"

	"github.com/labstack/echo/v5"
	"go.uber.org/zap"
)

// Pinger is the single capability the health check needs from the database;
// *pgxpool.Pool satisfies it.
type Pinger interface {
	Ping(ctx context.Context) error
}

// healthController holds the database handle the /health probe pings
type healthController struct {
	db Pinger
}

// NewHealthController wires the health endpoint to db, the database handle
// the probe will ping (the shared pgx pool in practice). db is stored as-is:
// a nil value isn't rejected here but panics on the first health request.
// The returned controller does nothing until Health registers its route.
func NewHealthController(db Pinger) *healthController {
	return &healthController{db: db}
}

// Health registers GET /health on e, the readiness probe behind the
// container healthcheck and CD gating. Every request pings the database
// with a 2-second budget and answers 200 "ok" when it responds, or
// 503 "db: <error>" when the ping fails or times out — so the endpoint
// flips unhealthy the moment the database becomes unreachable and back
// once it recovers.
//
// Deliberately not swagger-annotated: the spec's base path is the
// versioned API prefix, and this probe lives at the root, outside it.
func (hc *healthController) Health(e *echo.Echo) {
	e.GET("/health", func(c *echo.Context) error {
		log := logging.From(c.Request().Context())
		log.Debug("GET /health")

		ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
		defer cancel()
		if err := hc.db.Ping(ctx); err != nil {
			log.Error("database ping failed", zap.Error(err))
			return c.String(http.StatusServiceUnavailable, "db: "+err.Error())
		}
		return c.String(http.StatusOK, "ok")
	})
}
