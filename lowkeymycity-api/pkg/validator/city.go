package validator

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"

	"lowkeymycity/pkg/logging"
	"lowkeymycity/pkg/types"

	"go.uber.org/zap"
)

// SqlExecutor is the single sqlc-generated query Init needs: every
// canonical city label from the cities table. The generated *Queries
// satisfies it; tests substitute a fake.
type SqlExecutor interface {
	GetCities(ctx context.Context) ([]string, error)
}

// CityValidator answers "is this a real US city?" from an in-memory set
// loaded once at startup, maps free-form spellings to the canonical city
// label, and exports the loaded set as the JSON file the frontend
// autocomplete fetches.
type CityValidator interface {
	Init(ctx context.Context, sql SqlExecutor) error
	IsValid(city string) bool
	GetCityID(city string) (string, bool)
	Export(path string) error
}

// cityValidator implements CityValidator with a map from lowercased,
// trimmed spelling to canonical label, plus the labels as the query
// ordered them (population descending) for Export. Both are filled by
// Init and guarded by the embedded mutex.
type cityValidator struct {
	store  map[string]string
	labels []string
	log    types.Logger
	sync.Mutex
}

// NewCityValidator returns an empty validator: until Init succeeds it
// knows no cities, so IsValid says false and GetCityID returns ("", false)
// for every input.
func NewCityValidator(log types.Logger) CityValidator {
	return &cityValidator{log: log}
}

// Init loads every city label from the database through sql (must be
// non-nil — nil panics) and builds the in-memory lookup, keyed by
// lowercased, trimmed label. It returns nil once the validator is ready,
// or the database error unchanged — in which case whatever set was loaded
// before keeps serving. Meant to run once at startup; calling it again
// atomically replaces the whole set. Canceling ctx aborts the load with
// the context's error.
func (c *cityValidator) Init(ctx context.Context, sql SqlExecutor) error {
	log := logging.From(ctx)
	log.Debug("cityValidator.Init")

	labels, err := sql.GetCities(ctx)
	if err != nil {
		log.Error("loading cities from database", zap.Error(err))
		return err
	}

	c.Lock()
	defer c.Unlock()

	c.store = make(map[string]string, len(labels))
	for _, label := range labels {
		c.store[strings.ToLower(strings.TrimSpace(label))] = label
	}
	c.labels = labels

	c.log.Info("city validator loaded", zap.Int("cities", len(c.store)))
	return nil
}

// Export writes the loaded labels — in the query's population-descending
// order, the ranking the frontend autocomplete relies on — as a JSON
// array to path, the file GET /api/v1/cities serves. Must follow a
// successful Init: before one there is nothing to export and the call
// errors, as it does on an unwritable path.
func (c *cityValidator) Export(path string) error {
	c.Lock()
	defer c.Unlock()

	if len(c.labels) == 0 {
		return errors.New("no cities loaded: Export must follow a successful Init")
	}

	data, err := json.Marshal(c.labels)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		c.log.Error("writing cities export", zap.String("path", path), zap.Error(err))
		return err
	}

	c.log.Info("cities exported", zap.String("path", path), zap.Int("cities", len(c.labels)))
	return nil
}

// IsValid reports whether city names a known US city. Matching ignores
// case and surrounding whitespace, so "  new york, ny " is valid as long
// as "New York, NY" is seeded; anything unknown — or any input before Init
// has succeeded — is false. Safe for concurrent use.
func (c *cityValidator) IsValid(city string) bool {
	c.log.Debug("cityValidator.IsValid", zap.String("city", city))

	c.Lock()
	defer c.Unlock()

	_, ok := c.store[strings.ToLower(strings.TrimSpace(city))]
	return ok
}

// GetCityID maps any spelling of a city name (matched ignoring case and
// surrounding whitespace) to its canonical label exactly as stored in the
// cities table, returning the label and true. For an unknown city — or any
// input before Init has succeeded — it returns "" and false. Safe for
// concurrent use.
func (c *cityValidator) GetCityID(city string) (string, bool) {
	c.log.Debug("cityValidator.GetCityID", zap.String("city", city))

	c.Lock()
	defer c.Unlock()

	id, ok := c.store[strings.ToLower(strings.TrimSpace(city))]
	return id, ok
}
