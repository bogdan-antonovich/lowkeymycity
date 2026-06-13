package quiz

import (
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"

	"lowkeymycity/internal/testdb"
	"lowkeymycity/pkg/types"
	"lowkeymycity/pkg/validator"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestQuizServiceIntegration runs the quiz service against a real postgres
// (schema and cities seed from the live migrations) with a real validator;
// only the LLM's HTTP layer is faked. The scenarios are the same contract
// promises the unit tests check, but here the cache, the dedup key and the
// quiz ordering are enforced by the actual tables.
func TestQuizServiceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test needs docker")
	}
	pool := testdb.New(t)
	sql := New(pool)
	cv := validator.NewCityValidator(zap.NewNop())
	require.NoError(t, cv.Init(t.Context(), validator.New(pool)), "the validator loads from the seeded cities table")

	// service builds one quiz service whose LLM always replies with reply
	service := func(reply string) (QuizService, *atomic.Int64) {
		client, calls := fakeLLM(reply, http.StatusOK)
		return NewQuizService(client, sql, cv, "city-vibe-prompt", "verdict-prompt"), calls
	}
	mustJSON := func(v any) string {
		b, err := json.Marshal(v)
		require.NoError(t, err)
		return string(b)
	}

	t.Run("city questions are generated once, then served from the cache", func(t *testing.T) {
		generating, llm := service(mustJSON(portlandQuestions))

		// first request for the city: cold cache, the LLM writes the quiz
		q, err := generating.GetPersonalizedQuestions(t.Context(), "Portland, OR")
		require.NoError(t, err)
		assert.Equal(t, portlandQuestions, q)
		assert.NotZero(t, llm.Load(), "a cold cache means the LLM writes the quiz")

		// second request, messy spelling, on a service whose LLM was never
		// taught the questions: only the database can be answering
		cached, idleLLM := service("the LLM has nothing useful to say")
		q, err = cached.GetPersonalizedQuestions(t.Context(), "  porTLand, or  ")
		require.NoError(t, err)
		assert.Equal(t, portlandQuestions, q, "the cached quiz, found under the canonical label")
		assert.Zero(t, idleLLM.Load(), "a cached city must not touch the LLM")
	})

	t.Run("cached questions come back in quiz order, not insert order", func(t *testing.T) {
		// seed a city's cache by hand, positions deliberately inserted
		// backwards — "in quiz order" must mean position order
		for _, row := range []struct {
			pos     int32
			meaning string
			text    string
		}{
			{2, "pace", "second question"},
			{1, "climate", "first question"},
		} {
			_, err := pool.Exec(t.Context(),
				`INSERT INTO city_questions (city_label, position, meaning_id, text, options) VALUES ($1, $2, $3, $4, $5)`,
				"Seattle, WA", row.pos, row.meaning, row.text, `[{"id":"a","text":"an option"}]`)
			require.NoError(t, err)
		}
		svc, _ := service("")

		q, err := svc.GetPersonalizedQuestions(t.Context(), "Seattle, WA")

		require.NoError(t, err)
		require.Len(t, q, 2)
		assert.Equal(t, "climate", q[0].ID, "position 1 leads regardless of insert order")
		assert.Equal(t, "pace", q[1].ID)
	})

	t.Run("a city missing from the cities table is invalid", func(t *testing.T) {
		svc, llm := service("")

		q, err := svc.GetPersonalizedQuestions(t.Context(), "Atlantis, XX")

		require.ErrorIs(t, err, ErrInvalidCity)
		assert.Empty(t, q)
		assert.Zero(t, llm.Load())
	})

	t.Run("the match bank is empty until seeded, then served in quiz order", func(t *testing.T) {
		svc, llm := service("")

		// migrations create match_questions but seed nothing
		q, err := svc.GetPreStoredQuestions(t.Context())
		require.NoError(t, err, "an unseeded bank is not an error")
		assert.Empty(t, q)

		// seed backwards again: quiz order must come from position
		for i, mq := range matchBank {
			_, err := pool.Exec(t.Context(),
				`INSERT INTO match_questions (meaning_id, position, text, options) VALUES ($1, $2, $3, $4)`,
				mq.ID, int32(len(matchBank)-i), mq.Text, mustJSON(mq.Options))
			require.NoError(t, err)
		}
		q, err = svc.GetPreStoredQuestions(t.Context())
		require.NoError(t, err)
		require.Len(t, q, len(matchBank))
		assert.Equal(t, matchBank[1], q[0], "seeded with inverted positions, served back in position order")
		assert.Equal(t, matchBank[0], q[1])
		assert.Zero(t, llm.Load(), "the bank is pre-stored — the LLM has no business here")
	})

	t.Run("a verdict is stored once, replayed from the table, and order-sensitive", func(t *testing.T) {
		generating, _ := service(mustJSON(portlandVerdict))

		first, err := generating.GetResults(t.Context(), "city", "Portland, OR", quizAnswers)
		require.NoError(t, err)
		assert.NotEmpty(t, first.ID, "the stored verdict carries its permanent id")
		assert.Equal(t, wantResult(first.ID, portlandVerdict), first)

		// identical resubmission on an LLM that would answer garbage: only
		// the stored row can be coming back
		replaying, idleLLM := service("not even json")
		second, err := replaying.GetResults(t.Context(), "city", "Portland, OR", quizAnswers)
		require.NoError(t, err)
		assert.Equal(t, first, second, "same submission, same stored row, same id")
		assert.Zero(t, idleLLM.Load(), "a seen combination must not touch the LLM")

		// the same answers in reverse order are a different submission
		reversed := []Answer{quizAnswers[1], quizAnswers[0]}
		third, err := generating.GetResults(t.Context(), "city", "Portland, OR", reversed)
		require.NoError(t, err)
		assert.NotEqual(t, first.ID, third.ID, "reversed answers earn their own verdict")
	})

	t.Run("city mode forces the verdict's city to the submitted one", func(t *testing.T) {
		wandering := portlandVerdict
		wandering.City = "Asheville, NC"
		svc, _ := service(mustJSON(wandering))
		// fresh city so this is a new combination, not a replay from above
		answers := []Answer{{QuestionID: "q1", Question: "mountains?", Answer: "yes"}}

		res, err := svc.GetResults(t.Context(), "city", "Denver, CO", answers)

		require.NoError(t, err)
		assert.Equal(t, "Denver, CO", res.City, "city mode: the submitted city wins over the LLM's pick")
	})

	t.Run("match mode lets the LLM's pick stand", func(t *testing.T) {
		svc, _ := service(mustJSON(ashevilleVerdict))
		answers := []Answer{{QuestionID: "q1", Question: "hurry?", Answer: "never"}}

		res, err := svc.GetResults(t.Context(), "match", "Portland, OR", answers)

		require.NoError(t, err)
		assert.Equal(t, "Asheville, NC", res.City)
	})

	t.Run("concurrent identical submissions get the same stored row", func(t *testing.T) {
		// "Concurrent identical submissions are safe — both get the same
		// stored row." The results table backs this with a UNIQUE
		// combination; here real goroutines race on a real constraint.
		svc, _ := service(mustJSON(portlandVerdict))
		answers := []Answer{{QuestionID: "q1", Question: "race condition?", Answer: "hopefully not"}}

		const racers = 4
		results := make([]types.QuizResult, racers)
		errs := make([]error, racers)
		var wg sync.WaitGroup
		for i := range racers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				results[i], errs[i] = svc.GetResults(t.Context(), "city", "Portland, OR", answers)
			}()
		}
		wg.Wait()

		for i := range racers {
			require.NoError(t, errs[i], "racer %d should not see an error", i)
			assert.Equal(t, results[0], results[i], "racer %d should get the same stored row", i)
		}
	})
}
