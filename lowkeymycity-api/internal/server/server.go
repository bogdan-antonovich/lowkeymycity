// Package server assembles the production Echo instance: edge middleware
// (request id, zap access log, panic recovery, CORS allowlist, body limit,
// two-tier rate limiting), the request validator behind c.Validate, and
// the central error handler that keeps internals out of 5xx responses.
// Controllers register their routes on the result; everything HTTP-generic
// lives here so they stay thin.
package server

import (
	"errors"
	"net"
	"net/http"

	"lowkeymycity/pkg/logging"
	"lowkeymycity/pkg/types"
	cityvalidator "lowkeymycity/pkg/validator"

	govalidator "github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"go.uber.org/zap"
)

// The API lives under two nested prefixes, each attached exactly once in
// main as echo groups: APIRoot marks the route as API surface, V1 is the
// version inside it. Controllers register relative paths on the version
// group; only /health stays outside — it's the ops probe, not API
// surface. APIPrefix is the composition, for everything that needs the
// full path of a current-version route (the LLM rate-limit set, the docs
// page, tests).
const (
	APIRoot   = "/api"
	V1        = "/v1"
	APIPrefix = APIRoot + V1
)

// llmRoutes are the route paths whose handlers may end up paying for an
// LLM call; they run under the strict rate limit instead of the usual one.
var llmRoutes = map[string]bool{
	APIPrefix + "/quiz":        true,
	APIPrefix + "/quiz/result": true,
}

// maxBodyBytes caps request bodies. The largest legitimate body is a
// finished quiz (a dozen answers of a sentence each); 64KB is generous.
const maxBodyBytes = 64 << 10

// New builds the Echo instance the API runs on. allowOrigins is the CORS
// allowlist (exact origins, no wildcards); cities answers the
// city_exists validation tag; trustedProxies is the extra CIDR ranges the
// X-Forwarded-For walk skips as proxy hops (the CDN's edges, in
// production) on top of the always-trusted loopback, link-local and
// private ranges. The middleware chain, outermost first: request id,
// access log, panic recovery, CORS, body limit, then rate limiting —
// 15 req/min per IP on the LLM-backed quiz endpoints, a generous default
// everywhere else. The caller registers routes on the result and owns
// serving it.
func New(log types.Logger, cities cityvalidator.CityValidator, allowOrigins []string, trustedProxies []*net.IPNet) *echo.Echo {
	e := echo.New()
	// the API sits behind the site's reverse proxy (and a CDN in front of
	// that): the visitor's address arrives in X-Forwarded-For, not on the
	// socket, and the rate limiter is only per-visitor when every proxy
	// hop in the chain is recognized as one
	trust := make([]echo.TrustOption, 0, len(trustedProxies))
	for _, ipRange := range trustedProxies {
		trust = append(trust, echo.TrustIPRange(ipRange))
	}
	e.IPExtractor = echo.ExtractIPFromXFFHeader(trust...)
	e.Validator = NewRequestValidator(log, cities)
	e.HTTPErrorHandler = newErrorHandler(log)

	e.Use(middleware.RequestID())
	e.Use(scopeLoggerToRequest(log))
	e.Use(accessLog(log))
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: allowOrigins,
		AllowMethods: []string{http.MethodGet, http.MethodPost},
		AllowHeaders: []string{echo.HeaderContentType},
	}))
	e.Use(middleware.BodyLimit(maxBodyBytes))

	// the usual limiter: roomy enough that no human browsing the site ever
	// meets it, tight enough to blunt dumb floods
	e.Use(middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Skipper: func(c *echo.Context) bool { return llmRoutes[c.Path()] },
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
			Rate:  20,
			Burst: 40,
		}),
	}))
	// the strict limiter: every request here can cost OpenAI money, so a
	// visitor gets 15 a minute and not more
	e.Use(middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Skipper: func(c *echo.Context) bool { return !llmRoutes[c.Path()] },
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
			Rate:  0.25, // refills 15 tokens per minute
			Burst: 15,
		}),
	}))

	return e
}

// RequestValidator implements echo.Validator over go-playground/validator,
// with the project's city_exists tag answered by the city validator. A
// failed validation comes back as a 400 HTTPError carrying the field-level
// reasons, ready to be returned from a handler as-is.
type RequestValidator struct {
	validate *govalidator.Validate
	log      types.Logger
}

// NewRequestValidator builds the validator; cities must be the Init-ed
// CityValidator backing the city_exists tag.
func NewRequestValidator(log types.Logger, cities cityvalidator.CityValidator) *RequestValidator {
	v := govalidator.New()
	// registration only errors on an empty tag or nil func, neither
	// possible here
	_ = v.RegisterValidation("city_exists", func(fl govalidator.FieldLevel) bool {
		return cities.IsValid(fl.Field().String())
	})
	return &RequestValidator{validate: v, log: log}
}

// Validate checks i against its validate tags and shapes a failure into
// the 400 the client deserves. The failure's one Error log line is written
// by the calling handler, not here — echo's Validator interface carries no
// context, so only the handler can stamp the line with the request id.
func (rv *RequestValidator) Validate(i any) error {
	rv.log.Debug("RequestValidator.Validate", zap.Any("request", i))

	if err := rv.validate.Struct(i); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}

// scopeLoggerToRequest stamps every request's context with a logger that
// carries the request id, so each layer below — handlers, services, the
// LLM and storage helpers — writes lines that group by request. Must sit
// after RequestID, which decides the id.
func scopeLoggerToRequest(log types.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			scoped := logging.With(log, zap.String("request_id", c.Response().Header().Get(echo.HeaderXRequestID)))
			c.SetRequest(c.Request().WithContext(logging.Into(c.Request().Context(), scoped)))
			return next(c)
		}
	}
}

// accessLog emits the one Info line every request gets: id, peer, method,
// URI, status, latency, and the handler's error when there was one. This
// line is the production signal; handler-level Debug entries are the dev
// detail underneath it.
func accessLog(log types.Logger) echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogRequestID: true,
		LogRemoteIP:  true,
		LogMethod:    true,
		LogURI:       true,
		LogStatus:    true,
		LogLatency:   true,
		HandleError:  true, // the error still reaches the central handler for the response
		LogValuesFunc: func(c *echo.Context, v middleware.RequestLoggerValues) error {
			fields := []zap.Field{
				zap.String("request_id", v.RequestID),
				zap.String("remote_ip", v.RemoteIP),
				zap.String("method", v.Method),
				zap.String("uri", v.URI),
				zap.Int("status", v.Status),
				zap.Duration("latency", v.Latency),
			}
			if v.Error != nil {
				// the access line records the outcome; the error's own Error
				// log already happened at its source
				fields = append(fields, zap.String("error", v.Error.Error()))
			}
			log.Info("request", fields...)
			return nil
		},
	})
}

// newErrorHandler wraps echo's default handler — exposeError=false, so
// 5xx bodies never carry internals — with the one log this package owns:
// a recovered panic is an error initiated inside the middleware chain, so
// its single Error line, stack included, is written here. Every other
// error was already logged at its source and passes through silently.
func newErrorHandler(log types.Logger) echo.HTTPErrorHandler {
	fallback := echo.DefaultHTTPErrorHandler(false)
	return func(c *echo.Context, err error) {
		var panicked *middleware.PanicStackError
		if errors.As(err, &panicked) {
			logging.From(c.Request().Context()).
				Error("panic recovered", zap.Error(panicked.Err), zap.ByteString("stack", panicked.Stack))
		}
		fallback(c, err)
	}
}
