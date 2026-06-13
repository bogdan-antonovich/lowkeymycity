package quiz

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"lowkeymycity/pkg/types"

	"github.com/jackc/pgx/v5"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// the quiz for the one city these tests know about — used both as the
// cached rows the database hands back and as the array the fake LLM writes
var portlandQuestions = []Question{
	{
		ID:   "climate",
		Text: "eight months of drizzle: cozy or career-ending?",
		Options: []types.Option{
			{ID: "cozy", Text: "cozy, hand me the sad lamp"},
			{ID: "no", Text: "i would simply perish"},
		},
	},
	{
		ID:   "pace",
		Text: "your ideal friday night winds down at...",
		Options: []types.Option{
			{ID: "ten", Text: "ten, like a civilized person"},
			{ID: "sunrise", Text: "the question offends me"},
		},
	},
}

// cachedRows is portlandQuestions the way the database returns them, in
// quiz order.
func cachedRows() []GetCityQuestionsRow {
	rows := make([]GetCityQuestionsRow, 0, len(portlandQuestions))
	for _, q := range portlandQuestions {
		rows = append(rows, GetCityQuestionsRow{MeaningID: q.ID, Text: q.Text, Options: q.Options})
	}
	return rows
}

// fakeLLM builds an openai.Client whose every request is answered locally
// by the middleware — no network. status >= 400 makes every call fail;
// otherwise the assistant replies with exactly `content`, on both the chat
// completions and responses endpoints, so the test doesn't care which one
// the service uses. calls counts requests so tests can assert the LLM was
// left alone.
func fakeLLM(content string, status int) (client openai.Client, calls *atomic.Int64) {
	calls = new(atomic.Int64)
	client = openai.NewClient(
		option.WithAPIKey("test-key"),
		option.WithMaxRetries(0),
		option.WithMiddleware(func(req *http.Request, _ option.MiddlewareNext) (*http.Response, error) {
			calls.Add(1)

			var body []byte
			switch {
			case status >= 400:
				body = []byte(`{"error":{"message":"llm exploded","type":"server_error"}}`)
			case strings.Contains(req.URL.Path, "chat/completions"):
				body, _ = json.Marshal(map[string]any{
					"id": "chatcmpl-test", "object": "chat.completion", "created": 1, "model": "gpt-test",
					"choices": []map[string]any{{
						"index": 0, "finish_reason": "stop",
						"message": map[string]any{"role": "assistant", "content": content},
					}},
				})
			default: // responses API
				body, _ = json.Marshal(map[string]any{
					"id": "resp-test", "object": "response", "created_at": 1, "model": "gpt-test", "status": "completed",
					"output": []map[string]any{{
						"type": "message", "id": "msg-test", "status": "completed", "role": "assistant",
						"content": []map[string]any{{"type": "output_text", "text": content, "annotations": []any{}}},
					}},
				})
			}
			return &http.Response{
				StatusCode: status,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewReader(body)),
				Request:    req,
			}, nil
		}),
	)
	return client, calls
}

// stubValidator returns a CityValidator that knows exactly the cities in
// known (keyed by lowercased, trimmed spelling, valued by canonical
// label), however the service decides to ask it. Init stays unstubbed on
// purpose: the service has no business re-initializing its validator.
func stubValidator(ctrl *gomock.Controller, known map[string]string) *MockCityValidator {
	lookup := func(city string) (string, bool) {
		label, ok := known[strings.ToLower(strings.TrimSpace(city))]
		return label, ok
	}
	v := NewMockCityValidator(ctrl)
	v.EXPECT().IsValid(gomock.Any()).DoAndReturn(func(city string) bool {
		_, ok := lookup(city)
		return ok
	}).AnyTimes()
	v.EXPECT().GetCityID(gomock.Any()).DoAndReturn(lookup).AnyTimes()
	return v
}

var portlandOnly = map[string]string{"portland, or": "Portland, OR"}

// TestGetPersonalizedQuestions exercises the documented contract of
// GetPersonalizedQuestions: validator gates the city, the cache serves
// repeat requests, the LLM writes the quiz exactly once per city, and
// every failure surfaces unchanged.
func TestGetPersonalizedQuestions(t *testing.T) {
	t.Run("unknown city returns ErrInvalidCity", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		client, calls := fakeLLM("", http.StatusOK)
		// no expectations on sql: an invalid city must never reach storage
		svc := NewQuizService(client, NewMockSqlExecutor(ctrl), stubValidator(ctrl, portlandOnly), "city-vibe-prompt", "verdict-prompt")

		q, err := svc.GetPersonalizedQuestions(t.Context(), "Atlantis, XX")

		require.ErrorIs(t, err, ErrInvalidCity)
		assert.Empty(t, q, "no questions for an unknown city")
		assert.Zero(t, calls.Load(), "no LLM call for an unknown city")
	})

	t.Run("cached city is served from the cache in quiz order", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		client, calls := fakeLLM("", http.StatusOK)
		sql := NewMockSqlExecutor(ctrl)
		// the messy spelling below must reach the cache as the canonical label
		sql.EXPECT().GetCityQuestions(gomock.Any(), "Portland, OR").Return(cachedRows(), nil)
		svc := NewQuizService(client, sql, stubValidator(ctrl, portlandOnly), "city-vibe-prompt", "verdict-prompt")

		q, err := svc.GetPersonalizedQuestions(t.Context(), "  porTLand, or  ")

		require.NoError(t, err)
		assert.Equal(t, portlandQuestions, q, "cached questions, in quiz order")
		assert.Zero(t, calls.Load(), "a cached city must not touch the LLM")
	})

	t.Run("first request generates, caches and returns the quiz", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		reply, err := json.Marshal(portlandQuestions)
		require.NoError(t, err)
		client, calls := fakeLLM(string(reply), http.StatusOK)

		sql := NewMockSqlExecutor(ctrl)
		sql.EXPECT().GetCityQuestions(gomock.Any(), "Portland, OR").Return(nil, nil)
		var stored []AddCityQuestionParams
		sql.EXPECT().AddCityQuestion(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ any, arg AddCityQuestionParams) error {
				stored = append(stored, arg)
				return nil
			}).Times(len(portlandQuestions))
		svc := NewQuizService(client, sql, stubValidator(ctrl, portlandOnly), "city-vibe-prompt", "verdict-prompt")

		q, err := svc.GetPersonalizedQuestions(t.Context(), "Portland, OR")

		require.NoError(t, err)
		assert.Equal(t, portlandQuestions, q, "the freshly generated quiz comes back")
		assert.NotZero(t, calls.Load(), "a cold cache means the LLM writes the quiz")
		for i, arg := range stored {
			assert.Equal(t, "Portland, OR", arg.CityLabel, "cached under the canonical label")
			assert.Equal(t, portlandQuestions[i].ID, arg.MeaningID, "cached in slice order")
			assert.Equal(t, portlandQuestions[i].Text, arg.Text)
			assert.Equal(t, portlandQuestions[i].Options, arg.Options)
			if i > 0 {
				assert.Greater(t, arg.Position, stored[i-1].Position, "positions record quiz order")
			}
		}
	})

	t.Run("cache lookup failure comes back unchanged", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		client, calls := fakeLLM("", http.StatusOK)
		dbDown := errors.New("db: connection refused")
		sql := NewMockSqlExecutor(ctrl)
		sql.EXPECT().GetCityQuestions(gomock.Any(), "Portland, OR").Return(nil, dbDown)
		svc := NewQuizService(client, sql, stubValidator(ctrl, portlandOnly), "city-vibe-prompt", "verdict-prompt")

		q, err := svc.GetPersonalizedQuestions(t.Context(), "Portland, OR")

		require.ErrorIs(t, err, dbDown)
		assert.Empty(t, q)
		assert.Zero(t, calls.Load(), "a failed lookup is not an empty cache — no generation")
	})

	t.Run("generation failure comes back, nothing is cached", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		client, _ := fakeLLM("", http.StatusInternalServerError)
		sql := NewMockSqlExecutor(ctrl)
		sql.EXPECT().GetCityQuestions(gomock.Any(), "Portland, OR").Return(nil, nil)
		// no AddCityQuestion expectation: a failed generation must not write
		svc := NewQuizService(client, sql, stubValidator(ctrl, portlandOnly), "city-vibe-prompt", "verdict-prompt")

		q, err := svc.GetPersonalizedQuestions(t.Context(), "Portland, OR")

		require.Error(t, err)
		assert.Empty(t, q, "generation failed, so there are no questions to return")
	})

	t.Run("caching failure returns the questions alongside the error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		reply, err := json.Marshal(portlandQuestions)
		require.NoError(t, err)
		client, _ := fakeLLM(string(reply), http.StatusOK)

		insertBroke := errors.New("db: unique violation")
		sql := NewMockSqlExecutor(ctrl)
		sql.EXPECT().GetCityQuestions(gomock.Any(), "Portland, OR").Return(nil, nil)
		// every insert fails, whether the service stops at the first or not
		sql.EXPECT().AddCityQuestion(gomock.Any(), gomock.Any()).Return(insertBroke).MinTimes(1)
		svc := NewQuizService(client, sql, stubValidator(ctrl, portlandOnly), "city-vibe-prompt", "verdict-prompt")

		q, err := svc.GetPersonalizedQuestions(t.Context(), "Portland, OR")

		require.ErrorIs(t, err, insertBroke)
		assert.Equal(t, portlandQuestions, q, "generated questions come back even when caching fails")
	})
}

// matchBank is the fixed match-mode question bank these tests seed the
// fake database with.
var matchBank = []Question{
	{
		ID:   "social",
		Text: "a stranger says hi on the street. you...",
		Options: []types.Option{
			{ID: "hi", Text: "say hi back, obviously"},
			{ID: "flee", Text: "file a mental police report"},
		},
	},
	{
		ID:   "nature",
		Text: "how far away should the nearest tree be?",
		Options: []types.Option{
			{ID: "touching", Text: "i should be able to touch it"},
			{ID: "postcard", Text: "visible on a postcard is fine"},
		},
	},
}

// matchBankRows is matchBank the way the database returns it, in quiz order.
func matchBankRows() []GetMatchQuestionsRow {
	rows := make([]GetMatchQuestionsRow, 0, len(matchBank))
	for _, q := range matchBank {
		rows = append(rows, GetMatchQuestionsRow{MeaningID: q.ID, Text: q.Text, Options: q.Options})
	}
	return rows
}

// TestGetPreStoredQuestions exercises the documented contract of
// GetPreStoredQuestions: the bank is fixed and pre-stored — served from
// the database in quiz order, never generated — and the only documented
// failure is the database's own error, unchanged.
func TestGetPreStoredQuestions(t *testing.T) {
	t.Run("seeded bank comes back in quiz order", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		client, calls := fakeLLM("", http.StatusOK)
		sql := NewMockSqlExecutor(ctrl)
		sql.EXPECT().GetMatchQuestions(gomock.Any()).Return(matchBankRows(), nil)
		svc := NewQuizService(client, sql, stubValidator(ctrl, portlandOnly), "city-vibe-prompt", "verdict-prompt")

		q, err := svc.GetPreStoredQuestions(t.Context())

		require.NoError(t, err)
		assert.Equal(t, matchBank, q, "the stored bank, in quiz order")
		assert.Zero(t, calls.Load(), "the bank is pre-stored — the LLM has no business here")
	})

	t.Run("unseeded bank is an empty slice and a nil error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		client, _ := fakeLLM("", http.StatusOK)
		sql := NewMockSqlExecutor(ctrl)
		// faithful to the generated layer: a :many query with no rows hands
		// back a nil slice and no error
		sql.EXPECT().GetMatchQuestions(gomock.Any()).Return(nil, nil)
		svc := NewQuizService(client, sql, stubValidator(ctrl, portlandOnly), "city-vibe-prompt", "verdict-prompt")

		q, err := svc.GetPreStoredQuestions(t.Context())

		require.NoError(t, err, "an unseeded bank is not an error")
		assert.Empty(t, q)
		// the contract says "empty slice", not nil: a nil slice marshals to
		// JSON null and the frontend would see null where it expects []
		assert.NotNil(t, q, "contract promises an empty slice, not a nil one")
	})

	t.Run("database failure comes back unchanged, no questions", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		client, _ := fakeLLM("", http.StatusOK)
		dbDown := errors.New("db: connection refused")
		sql := NewMockSqlExecutor(ctrl)
		sql.EXPECT().GetMatchQuestions(gomock.Any()).Return(nil, dbDown)
		svc := NewQuizService(client, sql, stubValidator(ctrl, portlandOnly), "city-vibe-prompt", "verdict-prompt")

		q, err := svc.GetPreStoredQuestions(t.Context())

		require.ErrorIs(t, err, dbDown, "the database error must surface unchanged")
		assert.Empty(t, q)
	})
}

// quizAnswers is one finished quiz the way the controller hands it over —
// full question and answer text travel with the id.
var quizAnswers = []Answer{
	{QuestionID: "climate", Question: "eight months of drizzle: cozy or career-ending?", Answer: "cozy, hand me the sad lamp"},
	{QuestionID: "pace", Question: "your ideal friday night winds down at...", Answer: "ten, like a civilized person"},
}

// portlandVerdict is the id-less verdict the fake LLM writes for a city
// run — storage assigns the id later.
var portlandVerdict = types.QuizResult{
	Mode:         "city",
	City:         "Portland, OR",
	Score:        86,
	Title:        "portland is lowkey your soulmate",
	Summary:      "quiet streets, strong coffee, cancelable plans.",
	GreenFlags:   []string{"you and portland want the same things"},
	RedFlags:     []string{"the grey is real"},
	Alternatives: []types.Alternative{{City: "Bend, OR", Blurb: "the sunny-side alternative"}},
	Closing:      "visit once in february before you commit.",
}

// ashevilleVerdict is an id-less match-mode verdict where the LLM picked
// the city itself.
var ashevilleVerdict = types.QuizResult{
	Mode:         "match",
	City:         "Asheville, NC",
	Title:        "certified slow-living enjoyer",
	Summary:      "nobody there is in a hurry, and neither are you.",
	GreenFlags:   []string{"river arts district studios you can wander into"},
	RedFlags:     []string{"october is a siege"},
	Alternatives: []types.Alternative{{City: "Burlington, VT", Blurb: "the lake-town version"}},
	Closing:      "go in shoulder season.",
}

// rowFromVerdict is verdict v the way the results table stores it: under
// permanent id and deduplication combination.
func rowFromVerdict(id, comb string, v types.QuizResult) Result {
	return Result{
		ID: id, Combination: comb, Mode: v.Mode, City: v.City, Score: int32(v.Score),
		Title: v.Title, Summary: v.Summary, GreenFlags: v.GreenFlags, RedFlags: v.RedFlags,
		Alternatives: v.Alternatives, Closing: v.Closing,
	}
}

// rowFromParams is what the real SaveResult hands back: the saved row, now
// carrying its permanent id.
func rowFromParams(id string, arg SaveResultParams) Result {
	return Result{
		ID: id, Combination: arg.Combination, Mode: arg.Mode, City: arg.City, Score: arg.Score,
		Title: arg.Title, Summary: arg.Summary, GreenFlags: arg.GreenFlags, RedFlags: arg.RedFlags,
		Alternatives: arg.Alternatives, Closing: arg.Closing,
	}
}

// wantResult is verdict v as it must come back to the caller: unchanged,
// plus the permanent id storage assigned.
func wantResult(id string, v types.QuizResult) types.QuizResult {
	v.ID = id
	return v
}

// fakeResultsDB wires both result queries of the mock to one in-memory
// table so the dedup tests can watch state evolve across calls. It
// simulates the generated layer faithfully: a combination never saved is
// pgx.ErrNoRows, a save returns the row under a fresh res-N id.
type fakeResultsDB struct {
	rows  map[string]Result
	saves int
}

func primeResultsDB(sql *MockSqlExecutor) *fakeResultsDB {
	db := &fakeResultsDB{rows: map[string]Result{}}
	sql.EXPECT().GetResultByCombination(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, comb string) (Result, error) {
			row, ok := db.rows[comb]
			if !ok {
				return Result{}, pgx.ErrNoRows
			}
			return row, nil
		}).AnyTimes()
	sql.EXPECT().SaveResult(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, arg SaveResultParams) (Result, error) {
			db.saves++
			row := rowFromParams(fmt.Sprintf("res-%d", db.saves), arg)
			db.rows[arg.Combination] = row
			return row, nil
		}).AnyTimes()
	return db
}

// TestGetResults exercises the documented contract of GetResults: the
// submission is canonicalized into a JSON combination that deduplicates
// verdicts, the LLM writes each verdict exactly once, city mode forces the
// verdict's city while match mode keeps the LLM's pick, and every failure
// surfaces unchanged with a zero-valued result.
func TestGetResults(t *testing.T) {
	t.Run("a seen combination returns its stored result without touching the LLM", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		client, calls := fakeLLM("", http.StatusOK)
		sql := NewMockSqlExecutor(ctrl)
		// whatever the canonical combination looks like, the row stored under
		// it is the answer — and there is no SaveResult expectation: a replay
		// must not write
		sql.EXPECT().GetResultByCombination(gomock.Any(), gomock.Any()).
			Return(rowFromVerdict("res-stored", "{}", portlandVerdict), nil)
		svc := NewQuizService(client, sql, stubValidator(ctrl, portlandOnly), "city-vibe-prompt", "verdict-prompt")

		res, err := svc.GetResults(t.Context(), "city", "Portland, OR", quizAnswers)

		require.NoError(t, err)
		assert.Equal(t, wantResult("res-stored", portlandVerdict), res, "the previously stored verdict, id included")
		assert.Zero(t, calls.Load(), "a seen combination must not touch the LLM")
	})

	t.Run("a new combination has the LLM write the verdict and stores it under a fresh permanent id", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		reply, err := json.Marshal(portlandVerdict)
		require.NoError(t, err)
		client, calls := fakeLLM(string(reply), http.StatusOK)
		sql := NewMockSqlExecutor(ctrl)
		db := primeResultsDB(sql)
		svc := NewQuizService(client, sql, stubValidator(ctrl, portlandOnly), "city-vibe-prompt", "verdict-prompt")

		res, err := svc.GetResults(t.Context(), "city", "Portland, OR", quizAnswers)

		require.NoError(t, err)
		// "the returned QuizResult always carries the permanent id"
		assert.Equal(t, wantResult("res-1", portlandVerdict), res)
		assert.NotZero(t, calls.Load(), "a new combination means the LLM writes the verdict")
		require.Len(t, db.rows, 1, "exactly one verdict stored")
		for comb := range db.rows {
			// "canonicalized into a JSON combination"
			assert.True(t, json.Valid([]byte(comb)), "the dedup key should be JSON, got %q", comb)
		}
	})

	t.Run("an identical resubmission reuses the verdict — the LLM writes it exactly once", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		reply, err := json.Marshal(portlandVerdict)
		require.NoError(t, err)
		client, calls := fakeLLM(string(reply), http.StatusOK)
		sql := NewMockSqlExecutor(ctrl)
		db := primeResultsDB(sql)
		svc := NewQuizService(client, sql, stubValidator(ctrl, portlandOnly), "city-vibe-prompt", "verdict-prompt")

		first, err := svc.GetResults(t.Context(), "city", "Portland, OR", quizAnswers)
		require.NoError(t, err)
		llmCallsAfterFirst := calls.Load()

		second, err := svc.GetResults(t.Context(), "city", "Portland, OR", quizAnswers)

		require.NoError(t, err)
		assert.Equal(t, first, second, "both submissions get the same stored row")
		assert.Equal(t, llmCallsAfterFirst, calls.Load(), "the LLM generates the verdict exactly once")
		assert.Equal(t, 1, db.saves, "the verdict is stored exactly once")
	})

	t.Run("answer order is part of the submission's identity", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		reply, err := json.Marshal(portlandVerdict)
		require.NoError(t, err)
		client, _ := fakeLLM(string(reply), http.StatusOK)
		sql := NewMockSqlExecutor(ctrl)
		db := primeResultsDB(sql)
		svc := NewQuizService(client, sql, stubValidator(ctrl, portlandOnly), "city-vibe-prompt", "verdict-prompt")

		_, err = svc.GetResults(t.Context(), "city", "Portland, OR", quizAnswers)
		require.NoError(t, err)
		// "the answers in order" — the same answers reversed are a different
		// submission and must earn their own verdict
		reversed := []Answer{quizAnswers[1], quizAnswers[0]}
		_, err = svc.GetResults(t.Context(), "city", "Portland, OR", reversed)
		require.NoError(t, err)

		assert.Equal(t, 2, db.saves, "reversed answers are a new combination, not a replay")
		assert.Len(t, db.rows, 2, "two distinct combinations stored")
	})

	t.Run("city mode forces the verdict's city to the submitted one", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		// the LLM tries to wander: city-mode verdict naming a different city
		wandering := portlandVerdict
		wandering.City = "Asheville, NC"
		reply, err := json.Marshal(wandering)
		require.NoError(t, err)
		client, _ := fakeLLM(string(reply), http.StatusOK)
		sql := NewMockSqlExecutor(ctrl)
		db := primeResultsDB(sql)
		svc := NewQuizService(client, sql, stubValidator(ctrl, portlandOnly), "city-vibe-prompt", "verdict-prompt")

		res, err := svc.GetResults(t.Context(), "city", "Portland, OR", quizAnswers)

		require.NoError(t, err)
		assert.Equal(t, "Portland, OR", res.City, "city mode: the submitted city wins over the LLM's pick")
		// the stored row must carry the forced city too — a later identical
		// submission replays the stored row as-is
		for _, row := range db.rows {
			assert.Equal(t, "Portland, OR", row.City, "the forced city is what gets stored")
		}
	})

	t.Run("match mode lets the LLM's pick stand", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		reply, err := json.Marshal(ashevilleVerdict)
		require.NoError(t, err)
		client, _ := fakeLLM(string(reply), http.StatusOK)
		sql := NewMockSqlExecutor(ctrl)
		primeResultsDB(sql)
		svc := NewQuizService(client, sql, stubValidator(ctrl, portlandOnly), "city-vibe-prompt", "verdict-prompt")

		res, err := svc.GetResults(t.Context(), "match", "Portland, OR", quizAnswers)

		require.NoError(t, err)
		assert.Equal(t, "Asheville, NC", res.City, "match mode: the LLM's pick stands")
	})

	t.Run("database lookup failure comes back unchanged, result zero-valued", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		client, calls := fakeLLM("", http.StatusOK)
		dbDown := errors.New("db: connection refused")
		sql := NewMockSqlExecutor(ctrl)
		sql.EXPECT().GetResultByCombination(gomock.Any(), gomock.Any()).Return(Result{}, dbDown)
		svc := NewQuizService(client, sql, stubValidator(ctrl, portlandOnly), "city-vibe-prompt", "verdict-prompt")

		res, err := svc.GetResults(t.Context(), "city", "Portland, OR", quizAnswers)

		require.ErrorIs(t, err, dbDown, "the database error must surface unchanged")
		assert.Equal(t, types.QuizResult{}, res, "on failure the result is zero-valued")
		assert.Zero(t, calls.Load(), "a failed lookup is not a cache miss — no generation")
	})

	t.Run("LLM failure comes back, nothing is stored", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		client, _ := fakeLLM("", http.StatusInternalServerError)
		sql := NewMockSqlExecutor(ctrl)
		sql.EXPECT().GetResultByCombination(gomock.Any(), gomock.Any()).Return(Result{}, pgx.ErrNoRows)
		// no SaveResult expectation: a failed generation must not write
		svc := NewQuizService(client, sql, stubValidator(ctrl, portlandOnly), "city-vibe-prompt", "verdict-prompt")

		res, err := svc.GetResults(t.Context(), "city", "Portland, OR", quizAnswers)

		require.Error(t, err)
		assert.Equal(t, types.QuizResult{}, res, "on failure the result is zero-valued")
	})

	t.Run("an LLM reply that is not result JSON is an error, nothing is stored", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		// "an LLM reply that isn't valid result JSON" — prose instead of JSON
		client, _ := fakeLLM("the vibes are immaculate, no notes", http.StatusOK)
		sql := NewMockSqlExecutor(ctrl)
		sql.EXPECT().GetResultByCombination(gomock.Any(), gomock.Any()).Return(Result{}, pgx.ErrNoRows)
		// again no SaveResult expectation: garbage must not be stored
		svc := NewQuizService(client, sql, stubValidator(ctrl, portlandOnly), "city-vibe-prompt", "verdict-prompt")

		res, err := svc.GetResults(t.Context(), "city", "Portland, OR", quizAnswers)

		require.Error(t, err)
		assert.Equal(t, types.QuizResult{}, res, "on failure the result is zero-valued")
	})

	t.Run("a save failure loses the verdict and the next identical submission regenerates it", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		reply, err := json.Marshal(portlandVerdict)
		require.NoError(t, err)
		client, calls := fakeLLM(string(reply), http.StatusOK)
		insertBroke := errors.New("db: insert failed")
		sql := NewMockSqlExecutor(ctrl)
		// the combination is never stored, so both submissions miss the cache
		sql.EXPECT().GetResultByCombination(gomock.Any(), gomock.Any()).Return(Result{}, pgx.ErrNoRows).Times(2)
		gomock.InOrder(
			sql.EXPECT().SaveResult(gomock.Any(), gomock.Any()).Return(Result{}, insertBroke),
			sql.EXPECT().SaveResult(gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, arg SaveResultParams) (Result, error) {
					return rowFromParams("res-retry", arg), nil
				}),
		)
		svc := NewQuizService(client, sql, stubValidator(ctrl, portlandOnly), "city-vibe-prompt", "verdict-prompt")

		res, err := svc.GetResults(t.Context(), "city", "Portland, OR", quizAnswers)
		require.ErrorIs(t, err, insertBroke, "the save error must surface unchanged")
		assert.Equal(t, types.QuizResult{}, res, "the verdict is lost, not half-returned")
		llmCallsAfterFirst := calls.Load()

		res, err = svc.GetResults(t.Context(), "city", "Portland, OR", quizAnswers)

		require.NoError(t, err)
		assert.Equal(t, wantResult("res-retry", portlandVerdict), res)
		assert.Greater(t, calls.Load(), llmCallsAfterFirst, "the next identical submission regenerates the verdict")
	})
}
