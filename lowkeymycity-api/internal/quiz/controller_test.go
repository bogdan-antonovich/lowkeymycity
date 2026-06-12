package quiz

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"lowkeymycity/internal/server"
	"lowkeymycity/pkg/types"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

// newQuizAPI wires a quiz controller around the mocked service onto a
// fresh Echo, both routes registered and the request validator installed
// (it answers city_exists from the same stub the service tests use), ready
// for httptest traffic. The logger is a no-op: these tests assert
// responses, not log lines.
func newQuizAPI(t *testing.T, qs QuizService) *echo.Echo {
	e := echo.New()
	e.Validator = server.NewRequestValidator(zap.NewNop(), stubValidator(gomock.NewController(t), portlandOnly))
	qc := NewQuizController(qs)
	g := e.Group(server.APIPrefix)
	qc.GetQuestions(g)
	qc.GetResults(g)
	return e
}

// do fires one request at the API and hands back the recorder.
func do(e *echo.Echo, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// TestControllerGetQuestions exercises the documented contract of the
// GET /quiz endpoint: mode "match" serves the fixed bank, anything else —
// empty included — is city mode, and service failures surface as 500s.
// The handler itself only binds and shapes JSON; everything else is the
// service's problem, which is why every scenario is one mock expectation.
func TestControllerGetQuestions(t *testing.T) {
	t.Run("mode=match answers 200 with the fixed bank", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		qs := NewMockQuizService(ctrl)
		// no GetPersonalizedQuestions expectation: match mode must not
		// wander into the city path
		qs.EXPECT().GetPreStoredQuestions(gomock.Any()).Return(matchBank, nil)

		rec := do(newQuizAPI(t, qs), httptest.NewRequest(http.MethodGet, "/api/v1/quiz?mode=match", nil))

		require.Equal(t, http.StatusOK, rec.Code)
		var got GetQuestionsResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		assert.Equal(t, matchBank, got.Questions, "the bank comes back wrapped in the response envelope")
		assert.Empty(t, got.City, "match mode has no city to echo")
	})

	t.Run("mode=city answers 200 with the city's questions", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		qs := NewMockQuizService(ctrl)
		// the city travels to the service as submitted — resolving spellings
		// is the service's job, not the handler's
		qs.EXPECT().GetPersonalizedQuestions(gomock.Any(), "Portland, OR").Return(portlandQuestions, nil)

		rec := do(newQuizAPI(t, qs), httptest.NewRequest(http.MethodGet, "/api/v1/quiz?mode=city&city=Portland%2C+OR", nil))

		require.Equal(t, http.StatusOK, rec.Code)
		var got GetQuestionsResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		assert.Equal(t, portlandQuestions, got.Questions)
		assert.Equal(t, "Portland, OR", got.City, "the requested city is echoed back")
	})

	t.Run("a mode that is neither match nor city is a 400", func(t *testing.T) {
		// strict mode: anything outside the enum fails validation before
		// any work happens — no service expectations on purpose
		for name, target := range map[string]string{
			"empty mode":   "/api/v1/quiz?city=Portland%2C+OR",
			"unknown mode": "/api/v1/quiz?mode=banana&city=Portland%2C+OR",
		} {
			t.Run(name, func(t *testing.T) {
				qs := NewMockQuizService(gomock.NewController(t))

				rec := do(newQuizAPI(t, qs), httptest.NewRequest(http.MethodGet, target, nil))

				assert.Equal(t, http.StatusBadRequest, rec.Code)
			})
		}
	})

	t.Run("city mode without a city is a 400", func(t *testing.T) {
		qs := NewMockQuizService(gomock.NewController(t))

		rec := do(newQuizAPI(t, qs), httptest.NewRequest(http.MethodGet, "/api/v1/quiz?mode=city", nil))

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("an unknown city is a 400", func(t *testing.T) {
		// city_exists rejects it at the edge, so the service is never
		// consulted — no expectations
		qs := NewMockQuizService(gomock.NewController(t))

		rec := do(newQuizAPI(t, qs), httptest.NewRequest(http.MethodGet, "/api/v1/quiz?mode=city&city=Atlantis%2C+XX", nil))

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("a database or LLM failure surfaces as a 500", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		qs := NewMockQuizService(ctrl)
		qs.EXPECT().GetPreStoredQuestions(gomock.Any()).Return(nil, errors.New("db: connection refused"))

		rec := do(newQuizAPI(t, qs), httptest.NewRequest(http.MethodGet, "/api/v1/quiz?mode=match", nil))

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

// TestControllerGetResults exercises the documented contract of the
// POST /quiz/result endpoint: a bound body is delegated to the service
// as-is, success is a 200 with the stored QuizResult (permanent id
// included), an unparseable body is a 400, failures are 500s.
func TestControllerGetResults(t *testing.T) {
	// resultBody is the canonical valid POST body for these tests.
	resultBody := func(t *testing.T) *bytes.Reader {
		body, err := json.Marshal(GetResultsRequest{Mode: "city", City: "Portland, OR", Answers: quizAnswers})
		require.NoError(t, err)
		return bytes.NewReader(body)
	}

	t.Run("a finished quiz answers 200 with the stored result, id included", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		qs := NewMockQuizService(ctrl)
		// mode, city and answers must reach the service exactly as submitted
		qs.EXPECT().GetResults(gomock.Any(), "city", "Portland, OR", quizAnswers).
			Return(wantResult("res-1", portlandVerdict), nil)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/quiz/result", resultBody(t))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := do(newQuizAPI(t, qs), req)

		require.Equal(t, http.StatusOK, rec.Code)
		var got types.QuizResult
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		// "the id the frontend builds /r/{id} and the PDF link from"
		assert.Equal(t, wantResult("res-1", portlandVerdict), got)
	})

	t.Run("an unparseable body is a 400 and never reaches the service", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		// no expectations at all: binding fails before any delegation
		qs := NewMockQuizService(ctrl)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/quiz/result", bytes.NewReader([]byte("{not json")))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := do(newQuizAPI(t, qs), req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("a submission without answers is a 400", func(t *testing.T) {
		// min=1 on answers: an empty quiz cannot earn a verdict — again no
		// service expectations
		qs := NewMockQuizService(gomock.NewController(t))
		body, err := json.Marshal(GetResultsRequest{Mode: "city", City: "Portland, OR"})
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/quiz/result", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := do(newQuizAPI(t, qs), req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("a database or LLM failure surfaces as a 500", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		qs := NewMockQuizService(ctrl)
		qs.EXPECT().GetResults(gomock.Any(), "city", "Portland, OR", quizAnswers).
			Return(types.QuizResult{}, errors.New("llm exploded"))

		req := httptest.NewRequest(http.MethodPost, "/api/v1/quiz/result", resultBody(t))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := do(newQuizAPI(t, qs), req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}
