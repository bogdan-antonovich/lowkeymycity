package results

import (
	"context"
	"lowkeymycity/pkg/logging"
	"lowkeymycity/pkg/types"

	"go.uber.org/zap"
)

// ResultsService reads stored quiz results.
type ResultsService interface {
	GetResult(ctx context.Context, id string) (results types.QuizResult, err error)
}

// SqlExecutor is the single sqlc-generated query the results service
// needs: one stored result by id. The generated *Queries satisfies it;
// tests substitute a fake.
type SqlExecutor interface {
	GetStoredResult(ctx context.Context, id string) (GetStoredResultRow, error)
}

// resultsService implements ResultsService as a thin read layer over the
// generated queries.
type resultsService struct {
	sql SqlExecutor
}

// NewResultsService builds the results service around sql, the generated
// SqlExecutor over the live pool. sql is stored as-is — nil isn't rejected
// here but panics on first use.
func NewResultsService(sql SqlExecutor) ResultsService {
	return &resultsService{sql: sql}
}

// GetResult loads the stored quiz result with the given id, exactly as it
// was saved at quiz submission. When no row has that id — never issued, or
// cleaned up since — the error is pgx.ErrNoRows and the result is
// zero-valued; any other database error passes through unchanged.
// Canceling ctx aborts the lookup with the context's error.
func (rs *resultsService) GetResult(ctx context.Context, id string) (results types.QuizResult, err error) {
	log := logging.From(ctx)
	log.Debug("resultsService.GetResult", zap.String("id", id))

	stored, err := rs.sql.GetStoredResult(ctx, id)
	if err != nil {
		log.Error("loading stored result", zap.Error(err), zap.String("id", id))
		return
	}

	results = types.QuizResult{
		ID:           stored.ID,
		Mode:         stored.Mode,
		City:         stored.City,
		Score:        int(stored.Score),
		Title:        stored.Title,
		Summary:      stored.Summary,
		GreenFlags:   stored.GreenFlags,
		RedFlags:     stored.RedFlags,
		Alternatives: stored.Alternatives,
		Closing:      stored.Closing,
	}

	return
}
