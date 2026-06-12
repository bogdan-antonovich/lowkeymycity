// Package testdb spins up a throwaway postgres for integration tests: one
// container per call, the real goose migrations applied (cities seed
// included), torn down with the test. Tests that can't afford docker
// should be run with -short and skip before calling New.
package testdb

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// Conn is how an external process (the e2e binary) reaches the test
// database — the same coordinates main.go reads from the DB_* env vars.
type Conn struct {
	Host string
	Port int
	Name string
	User string
	Pass string
}

// New starts a postgres container, applies every migration's Up section in
// filename order, and returns a connected pool. Container and pool are
// cleaned up automatically when the test finishes.
func New(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, _ := NewWithConn(t)
	return pool
}

// NewWithConn is New plus the connection coordinates, for tests that hand
// the database to a separate process instead of (only) using the pool.
func NewWithConn(t *testing.T) (*pgxpool.Pool, Conn) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	conn := Conn{Name: "lowkeymycity_test", User: "test", Pass: "test"}
	ctr, err := postgres.Run(ctx, "postgres:17-alpine",
		postgres.WithDatabase(conn.Name),
		postgres.WithUsername(conn.User),
		postgres.WithPassword(conn.Pass),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("starting postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(ctr); err != nil {
			t.Logf("terminating postgres container: %v", err)
		}
	})

	if conn.Host, err = ctr.Host(ctx); err != nil {
		t.Fatalf("postgres container host: %v", err)
	}
	mapped, err := ctr.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatalf("postgres container port: %v", err)
	}
	conn.Port = int(mapped.Num())

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("postgres connection string: %v", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connecting to test postgres: %v", err)
	}
	t.Cleanup(pool.Close)

	for _, file := range migrationFiles(t) {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("reading migration %s: %v", file, err)
		}
		// the files are goose migrations; only the Up section gets applied
		up, _, _ := strings.Cut(string(raw), "-- +goose Down")
		if _, err := pool.Exec(ctx, up); err != nil {
			t.Fatalf("applying migration %s: %v", filepath.Base(file), err)
		}
	}
	return pool, conn
}

// migrationFiles lists the repository's migrations in apply order, located
// relative to this source file so tests can run from any directory.
func migrationFiles(t *testing.T) []string {
	t.Helper()
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate testdb.go to find the migrations directory")
	}
	dir := filepath.Join(filepath.Dir(self), "..", "..", "migrations")
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no migrations found in %s (err: %v)", dir, err)
	}
	sort.Strings(files)
	return files
}
