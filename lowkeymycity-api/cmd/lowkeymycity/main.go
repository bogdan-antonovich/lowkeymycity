package main

import (
	"context"
	"fmt"
	"lowkeymycity/internal/apidocs"
	"lowkeymycity/internal/cities"
	"lowkeymycity/internal/health"
	"lowkeymycity/internal/quiz"
	"lowkeymycity/internal/results"
	"lowkeymycity/internal/server"
	"lowkeymycity/pkg/logging"
	"lowkeymycity/pkg/validator"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config is everything the process reads from the environment. The two
// ",file" fields (OpenAIKey, DB.Pass) expect the variable to hold a path
// to a file containing the secret, docker-secrets style — not the secret
// itself. All fields except Mode are required.
type Config struct {
	Mode                      string `env:"MODE" envDefault:"dev"`
	Port                      int    `env:"PORT,notEmpty"`
	GPTModel                  string `env:"OPENAI_MODEL,notEmpty"`
	LowkeyCityVibeCheckPrompt string `env:"LOWKEY_CITY_VIBE_CHECK_PROMPT" envDefault:""`
	QuizResultsPrompt         string `env:"QUIZ_RESULTS_PROMPT" envDefault:""`

	// CORSOrigins is the comma-separated CORS allowlist; outside prod the
	// Vite dev server origin is appended automatically.
	CORSOrigins []string `env:"CORS_ORIGINS" envSeparator:"," envDefault:"https://lowkeymycity.com,https://www.lowkeymycity.com"`

	// CitiesFile is where the validator exports the city list JSON that
	// GET /api/v1/cities serves to the frontend autocomplete.
	CitiesFile string `env:"CITIES_FILE" envDefault:"/tmp/cities.json"`

	// TrustedProxyRanges is the comma-separated CIDR list of proxy hops
	// the X-Forwarded-For walk skips when extracting the visitor's IP —
	// in production, Cloudflare's published ranges. Empty means only the
	// built-in loopback/link-local/private ranges are trusted.
	TrustedProxyRanges []string `env:"TRUSTED_PROXY_RANGES" envSeparator:"," envDefault:""`

	OpenAIKey string `env:"OPENAI_API_KEY,file,notEmpty"`
	DB        struct {
		Host string `env:"HOST,notEmpty"`
		Port int    `env:"PORT,notEmpty"`
		Name string `env:"NAME,notEmpty"`
		User string `env:"USER,notEmpty"`
		Pass string `env:"PASS,file,notEmpty"`
	} `envPrefix:"DB_"`
}

// newLogger builds the process logger, writing to stdout only — in
// production the Loki agent ships container output from there. mode "prod"
// selects JSON encoding at info level (machine-parseable, no ANSI colors);
// every other value, empty included, selects colored console encoding at
// debug level for development. The returned AtomicLevel can flip the
// level at runtime. Never fails.
func newLogger(mode string) (*zap.Logger, zap.AtomicLevel) {
	atom := zap.NewAtomicLevel()

	encCfg := zapcore.EncoderConfig{
		TimeKey:      "timestamp",
		LevelKey:     "level",
		MessageKey:   "message",
		CallerKey:    "caller",
		EncodeCaller: zapcore.ShortCallerEncoder,
		EncodeTime:   zapcore.ISO8601TimeEncoder,
	}

	var enc zapcore.Encoder
	if mode == "prod" {
		atom.SetLevel(zapcore.InfoLevel)
		// JSON without ANSI colors, so Loki can parse the lines
		encCfg.EncodeLevel = zapcore.LowercaseLevelEncoder
		enc = zapcore.NewJSONEncoder(encCfg)
	} else {
		atom.SetLevel(zapcore.DebugLevel)
		encCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		enc = zapcore.NewConsoleEncoder(encCfg)
	}

	core := zapcore.NewCore(enc, zapcore.AddSync(os.Stdout), atom)
	return zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)), atom
}

// main boots the API: parse the environment into Config, build the
// logger, connect to and ping postgres (10s budget — failure kills the
// boot), load the city validator from the cities table, then wire the
// quiz, results and health routes onto Echo. Every wiring failure panics:
// the process either comes up whole or not at all.
//
// @title        LowkeyMyCity API
// @version      1.0
// @description  Backend for the lowkeymycity.com city vibe-check quiz.
// @BasePath     /api/v1
func main() {
	cfg := Config{}
	err := env.Parse(&cfg)
	if err != nil {
		panic(err)
	}

	logger, _ := newLogger(cfg.Mode)
	defer func() { _ = logger.Sync() }()
	// the logger logging.From hands out wherever the context carries no
	// request-scoped one
	logging.SetDefault(logger)

	dsn := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.DB.User, cfg.DB.Pass),
		Host:   fmt.Sprintf("%s:%d", cfg.DB.Host, cfg.DB.Port),
		Path:   cfg.DB.Name,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn.String())
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		panic(err)
	}
	logger.Info("postgres connected", zap.String("host", cfg.DB.Host), zap.String("database", cfg.DB.Name))

	cityValidator := validator.NewCityValidator(logger)
	if err := cityValidator.Init(ctx, validator.New(pool)); err != nil {
		panic(err)
	}
	if err := cityValidator.Export(cfg.CitiesFile); err != nil {
		panic(err)
	}

	openaiClient := openai.NewClient(
		option.WithAPIKey(cfg.OpenAIKey),
	)

	origins := cfg.CORSOrigins
	if cfg.Mode != "prod" {
		origins = append(origins, "https://lowkeymycity.com", "https://www.lowkeymycity.com")
	}

	// a typo'd CIDR silently un-trusting the CDN would break per-visitor
	// rate limiting, so a bad range kills the boot instead
	trustedProxies := make([]*net.IPNet, 0, len(cfg.TrustedProxyRanges))
	for _, cidr := range cfg.TrustedProxyRanges {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" {
			continue
		}
		_, ipRange, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Sprintf("TRUSTED_PROXY_RANGES entry %q: %v", cidr, err))
		}
		trustedProxies = append(trustedProxies, ipRange)
	}

	e := server.New(logger, cityValidator, origins, trustedProxies)
	// the one place the prefixes are attached: /api marks the surface,
	// /v1 versions it, and every controller below registers relative
	// paths on the version group
	api := e.Group(server.APIRoot).Group(server.V1)

	quizService := quiz.NewQuizService(openaiClient, quiz.New(pool), cityValidator, cfg.LowkeyCityVibeCheckPrompt, cfg.QuizResultsPrompt)
	quizController := quiz.NewQuizController(quizService)
	quizController.GetQuestions(api)
	quizController.GetResults(api)

	resultsService := results.NewResultsService(results.New(pool))
	resultsController := results.NewResultsController(resultsService)
	resultsController.GetResult(api)
	resultsController.GetPDF(api)

	healthController := health.NewHealthController(pool)
	healthController.Health(e)

	docsController := apidocs.NewDocsController()
	docsController.UI(api)

	citiesController := cities.NewCitiesController(cfg.CitiesFile)
	citiesController.GetCities(api)

	logger.Info("api wired — starting server", zap.Int("port", cfg.Port), zap.String("mode", cfg.Mode))
	if err := e.Start(fmt.Sprintf(":%d", cfg.Port)); err != nil {
		panic(err)
	}
}
