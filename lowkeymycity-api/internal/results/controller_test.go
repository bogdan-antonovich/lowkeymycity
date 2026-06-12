package results

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"lowkeymycity/internal/server"
	"lowkeymycity/pkg/types"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// newResultsAPI wires a results controller around the mocked service onto
// a fresh Echo, both routes registered, ready for httptest traffic. The
// logger is a no-op: these tests assert responses, not log lines.
func newResultsAPI(rs ResultsService) *echo.Echo {
	e := echo.New()
	rc := NewResultsController(rs)
	g := e.Group(server.APIPrefix)
	rc.GetResult(g)
	rc.GetPDF(g)
	return e
}

// do fires one request at the API and hands back the recorder.
func do(e *echo.Echo, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// TestControllerGetResult exercises the documented contract of the
// GET /results/:id endpoint: the stored verdict as a 200 JSON body, and —
// explicitly documented as current behavior — a 500 for an unknown id,
// since the no-rows case is not yet mapped to 404.
func TestControllerGetResult(t *testing.T) {
	t.Run("an issued id answers 200 with the full stored result", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		rs := NewMockResultsService(ctrl)
		// the path parameter must reach the service untouched
		rs.EXPECT().GetResult(gomock.Any(), "res-1").Return(storedVerdict, nil)

		rec := do(newResultsAPI(rs), httptest.NewRequest(http.MethodGet, "/api/v1/results/res-1", nil))

		require.Equal(t, http.StatusOK, rec.Code)
		var got types.QuizResult
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		assert.Equal(t, storedVerdict, got, "the shareable page gets the verdict exactly as stored")
	})

	t.Run("an unknown id is a 404", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		rs := NewMockResultsService(ctrl)
		rs.EXPECT().GetResult(gomock.Any(), "res-ghost").Return(types.QuizResult{}, pgx.ErrNoRows)

		rec := do(newResultsAPI(rs), httptest.NewRequest(http.MethodGet, "/api/v1/results/res-ghost", nil))

		// pgx.ErrNoRows is the documented "never issued" sentinel — a dead
		// share link is the visitor's 404, not the server's 500
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("a database failure surfaces as a 500", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		rs := NewMockResultsService(ctrl)
		rs.EXPECT().GetResult(gomock.Any(), "res-1").Return(types.QuizResult{}, errors.New("db: connection refused"))

		rec := do(newResultsAPI(rs), httptest.NewRequest(http.MethodGet, "/api/v1/results/res-1", nil))

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

// TestControllerGetPDF exercises the documented contract of the
// GET /results/:id/pdf endpoint: the stored verdict rendered as a PDF,
// served as an attachment named lowkeymycity.pdf; unknown ids and database
// failures are 500s. Rendering itself is pkg/pdf's business and has its
// own tests — here it only matters that real PDF bytes leave the handler.
func TestControllerGetPDF(t *testing.T) {
	t.Run("an issued id answers 200 with a PDF attachment", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		rs := NewMockResultsService(ctrl)
		rs.EXPECT().GetResult(gomock.Any(), "res-1").Return(storedVerdict, nil)

		rec := do(newResultsAPI(rs), httptest.NewRequest(http.MethodGet, "/api/v1/results/res-1/pdf", nil))

		require.Equal(t, http.StatusOK, rec.Code)
		assert.True(t, strings.HasPrefix(rec.Body.String(), "%PDF-"), "the body is a PDF document")
		// "browsers save it as lowkeymycity.pdf" — that name travels in the
		// attachment Content-Disposition
		disposition := rec.Header().Get(echo.HeaderContentDisposition)
		assert.Contains(t, disposition, "attachment")
		assert.Contains(t, disposition, "lowkeymycity.pdf")
		assert.Contains(t, rec.Header().Get(echo.HeaderContentType), "application/pdf")
	})

	t.Run("an unknown id is a 404", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		rs := NewMockResultsService(ctrl)
		rs.EXPECT().GetResult(gomock.Any(), "res-ghost").Return(types.QuizResult{}, pgx.ErrNoRows)

		rec := do(newResultsAPI(rs), httptest.NewRequest(http.MethodGet, "/api/v1/results/res-ghost/pdf", nil))

		// same mapping as the JSON endpoint — the two routes must agree on
		// what a dead link means
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("a database failure surfaces as a 500", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		rs := NewMockResultsService(ctrl)
		rs.EXPECT().GetResult(gomock.Any(), "res-1").Return(types.QuizResult{}, errors.New("db: connection refused"))

		rec := do(newResultsAPI(rs), httptest.NewRequest(http.MethodGet, "/api/v1/results/res-1/pdf", nil))

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}
