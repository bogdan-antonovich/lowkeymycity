// Package e2e boots the real thing: the actual lowkeymycity binary,
// compiled from cmd/lowkeymycity and configured exactly like production —
// env vars, docker-secret-style key files, a real postgres with the real
// migrations. The single substitution is OPENAI_BASE_URL, which the
// openai SDK honors natively, pointed at a local fake — OpenAI itself is
// the one thing we trust to work. Everything else (echo middleware,
// validation, rate limits, sqlc, the validator, PDF rendering) runs in
// the same process topology it ships in.
//
// Run locally with: go test ./e2e -v   (needs docker; skipped by -short)
package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"lowkeymycity/internal/testdb"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// questionsLLMReply is what the fake LLM writes when asked for a city quiz —
// options are {id,text} objects because that is what the LLM prompt produces.
const questionsLLMReply = `[
	{"id":"climate","text":"eight months of drizzle: cozy or career-ending?","options":[
		{"id":"cozy","text":"cozy, hand me the sad lamp"},{"id":"no","text":"i would simply perish"}]},
	{"id":"pace","text":"your ideal friday night winds down at...","options":[
		{"id":"ten","text":"ten, like a civilized person"},{"id":"sunrise","text":"the question offends me"}]}
]`

// questionsJSON is the shape the API serves to the frontend — options are
// flattened to plain text strings (the {id,text} layer is stripped in the
// controller before the response goes out).
const questionsJSON = `[
	{"id":"climate","text":"eight months of drizzle: cozy or career-ending?","options":[
		"cozy, hand me the sad lamp","i would simply perish"]},
	{"id":"pace","text":"your ideal friday night winds down at...","options":[
		"ten, like a civilized person","the question offends me"]}
]`

// verdictJSON is the fake LLM's verdict for a finished quiz — deliberately
// naming the wrong city, so the city-forcing rule is visible end to end.
const verdictJSON = `{
	"id":"","mode":"city","city":"Asheville, NC","score":86,
	"title":"portland is lowkey your soulmate",
	"summary":"quiet streets, strong coffee, cancelable plans.",
	"greenFlags":["you and portland want the same things"],
	"redFlags":["the grey is real"],
	"alternatives":[{"city":"Bend, OR","blurb":"the sunny-side alternative"}],
	"closing":"visit once in february before you commit."
}`

// submissionJSON is one finished quiz as the frontend posts it.
const submissionJSON = `{
	"mode":"city","city":"Portland, OR",
	"answers":[
		{"questionId":"climate","question":"eight months of drizzle: cozy or career-ending?","answer":"cozy, hand me the sad lamp"},
		{"questionId":"pace","question":"your ideal friday night winds down at...","answer":"ten, like a civilized person"}
	]
}`

// fakeLLM impersonates the OpenAI chat-completions endpoint over real
// HTTP. The assistant's next reply is settable between test steps, which
// is how one fake serves both question generation and verdicts.
type fakeLLM struct {
	mu  sync.Mutex
	msg string
	srv *httptest.Server
}

func newFakeLLM(t *testing.T) *fakeLLM {
	t.Helper()
	f := &fakeLLM{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			http.NotFound(w, r)
			return
		}
		f.mu.Lock()
		msg := f.msg
		f.mu.Unlock()
		body, _ := json.Marshal(map[string]any{
			"id": "chatcmpl-e2e", "object": "chat.completion", "created": 1, "model": "gpt-test",
			"choices": []map[string]any{{
				"index": 0, "finish_reason": "stop",
				"message": map[string]any{"role": "assistant", "content": msg},
			}},
		})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

// replyWith sets what the assistant says to every following LLM call.
func (f *fakeLLM) replyWith(msg string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.msg = msg
}

// logWriter pipes the api process's output into the test log, so
// `go test -v` shows the same lines production would ship to Loki.
type logWriter struct{ t *testing.T }

func (w logWriter) Write(p []byte) (int, error) {
	w.t.Logf("api | %s", bytes.TrimRight(p, "\n"))
	return len(p), nil
}

// secretFile writes value into a temp file and returns its path —
// docker-secrets style, the way the ",file" env fields expect it.
func secretFile(t *testing.T, name, value string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(value), 0o600))
	return path
}

// startAPI compiles cmd/lowkeymycity and launches it configured like prod:
// DB_* pointing at the test postgres, secrets in files, the LLM redirected
// via OPENAI_BASE_URL. It blocks until /health answers ok and returns the
// base URL. The process dies with the test.
func startAPI(t *testing.T, db testdb.Conn, llmURL string) string {
	t.Helper()

	_, self, _, ok := runtime.Caller(0)
	require.True(t, ok, "cannot locate e2e_test.go to find the module root")
	root := filepath.Join(filepath.Dir(self), "..")

	bin := filepath.Join(t.TempDir(), "lowkeymycity")
	build := exec.Command("go", "build", "-o", bin, "./cmd/lowkeymycity")
	build.Dir = root
	out, err := build.CombinedOutput()
	require.NoError(t, err, "building the binary: %s", out)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := ln.Addr().(*net.TCPAddr).Port
	require.NoError(t, ln.Close())

	api := exec.Command(bin)
	api.Env = append(os.Environ(),
		"MODE=dev",
		fmt.Sprintf("PORT=%d", port),
		"OPENAI_MODEL=gpt-test",
		"OPENAI_API_KEY="+secretFile(t, "openai_key", "e2e-test-key"),
		"OPENAI_BASE_URL="+llmURL,
		"DB_HOST="+db.Host,
		fmt.Sprintf("DB_PORT=%d", db.Port),
		"DB_NAME="+db.Name,
		"DB_USER="+db.User,
		"DB_PASS="+secretFile(t, "db_pass", db.Pass),
		"CITIES_FILE="+filepath.Join(t.TempDir(), "cities.json"),
	)
	api.Stdout = logWriter{t}
	api.Stderr = logWriter{t}
	require.NoError(t, api.Start(), "starting the api binary")
	t.Cleanup(func() {
		_ = api.Process.Kill()
		_ = api.Wait()
	})

	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(base + "/health")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return base
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("the api never became healthy")
	return ""
}

// get fires a GET and returns the response with its body read.
func get(t *testing.T, url string) (*http.Response, string) {
	t.Helper()
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp, string(body)
}

// questionsOf unwraps the GET /quiz response envelope and returns the
// questions array as raw JSON.
func questionsOf(t *testing.T, body string) string {
	t.Helper()
	var envelope struct {
		Questions json.RawMessage `json:"questions"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &envelope))
	return string(envelope.Questions)
}

// post fires a JSON POST and returns the response with its body read.
func post(t *testing.T, url, body string) (*http.Response, string) {
	t.Helper()
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	out, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp, string(out)
}

// TestE2E walks the whole product story against the real binary: boot,
// generate-and-cache a city quiz, validate input at the edge, turn a
// submission into a permanent shareable result (JSON and PDF), and confirm
// the protective middleware — CORS, request ids, the 15/min LLM budget —
// behaves. Subtests share one process and run in order; the rate-limit one
// must stay last because it spends the budget.
func TestE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e needs docker and a compiled binary")
	}

	_, db := testdb.NewWithConn(t)
	llm := newFakeLLM(t)
	base := startAPI(t, db, llm.srv.URL)

	t.Run("the binary boots and reports healthy", func(t *testing.T) {
		resp, body := get(t, base+"/health")
		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "ok", body)
	})

	t.Run("the exported city list is served, biggest city first", func(t *testing.T) {
		resp, body := get(t, base+"/api/v1/cities")
		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Type"), "application/json")
		assert.Equal(t, "public, max-age=86400", resp.Header.Get("Cache-Control"))

		var cities []string
		require.NoError(t, json.Unmarshal([]byte(body), &cities))
		require.NotEmpty(t, cities)
		// the export orders by population: the seed's biggest city leads
		assert.Equal(t, "New York, NY", cities[0])
		assert.Contains(t, cities, "Portland, OR")
	})

	t.Run("a city quiz is generated once, then served from the cache", func(t *testing.T) {
		llm.replyWith(questionsLLMReply)
		resp, body := get(t, base+"/api/v1/quiz?mode=city&city=+porTLand%2C+or+")
		require.Equal(t, http.StatusOK, resp.StatusCode, "body: %s", body)
		assert.JSONEq(t, questionsJSON, questionsOf(t, body), "the LLM-written quiz comes back")

		// the LLM now talks garbage: only the database can answer correctly
		llm.replyWith("the llm has left the chat")
		resp, body = get(t, base+"/api/v1/quiz?mode=city&city=Portland%2C+OR")
		require.Equal(t, http.StatusOK, resp.StatusCode, "body: %s", body)
		assert.JSONEq(t, questionsJSON, questionsOf(t, body), "second request is served from the cache")
		assert.Contains(t, body, `"city":"Portland, OR"`, "the requested city is echoed back")
	})

	t.Run("the match bank is an empty JSON array, not null", func(t *testing.T) {
		resp, body := get(t, base+"/api/v1/quiz?mode=match")
		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.JSONEq(t, "[]", questionsOf(t, body), "an unseeded bank serializes as [] for the frontend")
	})

	t.Run("validation guards the edge with 400s", func(t *testing.T) {
		for name, target := range map[string]string{
			"unknown mode": "/api/v1/quiz?mode=banana&city=Portland%2C+OR",
			"missing city": "/api/v1/quiz?mode=city",
			"unknown city": "/api/v1/quiz?mode=city&city=Atlantis%2C+XX",
		} {
			resp, _ := get(t, base+target)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "%s should be a 400", name)
		}
	})

	t.Run("a finished quiz becomes a permanent, shareable result", func(t *testing.T) {
		// the verdict names Asheville; city mode must force Portland
		llm.replyWith(verdictJSON)
		resp, body := post(t, base+"/api/v1/quiz/result", submissionJSON)
		require.Equal(t, http.StatusOK, resp.StatusCode, "body: %s", body)

		var result struct {
			ID   string `json:"id"`
			City string `json:"city"`
		}
		require.NoError(t, json.Unmarshal([]byte(body), &result))
		require.NotEmpty(t, result.ID, "the verdict carries its permanent id")
		assert.Equal(t, "Portland, OR", result.City, "city mode forces the submitted city")

		// resubmit with a garbage LLM: only the stored row can come back
		llm.replyWith("not even json")
		resp, replay := post(t, base+"/api/v1/quiz/result", submissionJSON)
		require.Equal(t, http.StatusOK, resp.StatusCode, "body: %s", replay)
		assert.JSONEq(t, body, replay, "identical submission replays the stored verdict, id included")

		resp, shared := get(t, base+"/api/v1/results/"+result.ID)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.JSONEq(t, body, shared, "the share page reads the same stored verdict")

		resp, pdf := get(t, base+"/api/v1/results/"+result.ID+"/pdf")
		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.True(t, strings.HasPrefix(pdf, "%PDF-"), "the receipt is a real PDF")
		assert.Contains(t, resp.Header.Get("Content-Disposition"), "lowkeymycity.pdf")

		resp, _ = get(t, base+"/api/v1/results/res-ghost")
		assert.Equal(t, http.StatusNotFound, resp.StatusCode, "a dead share link is a 404")
	})

	t.Run("CORS answers the allowlist and every response is traceable", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodOptions, base+"/api/v1/quiz", nil)
		require.NoError(t, err)
		req.Header.Set("Origin", "https://lowkeymycity.com")
		req.Header.Set("Access-Control-Request-Method", http.MethodGet)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		_ = resp.Body.Close()
		assert.Equal(t, "https://lowkeymycity.com", resp.Header.Get("Access-Control-Allow-Origin"))

		health, _ := get(t, base+"/health")
		assert.NotEmpty(t, health.Header.Get("X-Request-Id"), "every response carries a request id")
	})

	// keep this last: it deliberately exhausts the LLM budget
	t.Run("the LLM endpoints run out at 15 per minute, the rest keep working", func(t *testing.T) {
		denied := false
		for i := 0; i < 25 && !denied; i++ {
			resp, _ := get(t, base+"/api/v1/quiz?mode=match")
			denied = resp.StatusCode == http.StatusTooManyRequests
		}
		assert.True(t, denied, "hammering the quiz endpoint must eventually answer 429")

		resp, _ := get(t, base+"/health")
		assert.Equal(t, http.StatusOK, resp.StatusCode, "non-LLM endpoints are not starved by the strict bucket")
	})
}
