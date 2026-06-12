package quiz

import (
	"context"
	"errors"
	"fmt"
	"lowkeymycity/pkg/logging"
	"lowkeymycity/pkg/types"
	"lowkeymycity/pkg/validator"

	"github.com/bytedance/sonic"
	"github.com/jackc/pgx/v5"
	"github.com/openai/openai-go/v3"
	"go.uber.org/zap"
)

// InvalidCityError is returned when a city name is not in the validator's
// set of known US cities.
var InvalidCityError = errors.New("invalid city")

// QuizService produces quiz questions and turns finished quizzes into
// stored results.
type QuizService interface {
	GetPersonalizedQuestions(ctx context.Context, city string) (q []Question, err error)
	GetPreStoredQuestions(ctx context.Context) (q []Question, err error)
	GetResults(ctx context.Context, mode, city string, answers []Answer) (results types.QuizResult, err error)
}

// SqlExecutor is the slice of the sqlc-generated query API the quiz
// service needs: the per-city question cache, the match-mode question
// bank, and result storage keyed by answer combination. The generated
// *Queries satisfies it; tests substitute a fake.
type SqlExecutor interface {
	GetCityQuestions(ctx context.Context, cityLabel string) ([]GetCityQuestionsRow, error)
	GetMatchQuestions(ctx context.Context) ([]GetMatchQuestionsRow, error)
	GetResultByCombination(ctx context.Context, combination string) (Result, error)
	SaveResult(ctx context.Context, arg SaveResultParams) (Result, error)
	AddCityQuestion(ctx context.Context, arg AddCityQuestionParams) error
}

// quizService implements QuizService: the LLM client writes questions and
// verdicts, sql caches and stores them, validator gates city input.
type quizService struct {
	client                    openai.Client
	sql                       SqlExecutor
	validator                 validator.CityValidator
	lowkeyCityVibeCheckPrompt string
	quizeResultPrompt         string
}

// Question is one quiz question as served to the frontend. ID names the
// vibe axis the question measures (climate, pace, ...) and is stable
// across modes.
type Question struct {
	ID      string         `json:"id"`
	Text    string         `json:"text"`
	Options []types.Option `json:"options"`
}

// NewQuizService assembles the quiz service from its three dependencies:
// an authenticated OpenAI client, the generated SqlExecutor over the live
// pool, and an Init-ed CityValidator. Nothing is checked here — a nil sql
// or validator panics on first use, a misconfigured client fails on the
// first LLM call.
func NewQuizService(client openai.Client, sql SqlExecutor, validator validator.CityValidator, lowkeyCityVibeCheckPrompt, quizeResultPrompt string) *quizService {
	return &quizService{client: client, sql: sql, validator: validator, lowkeyCityVibeCheckPrompt: lowkeyCityVibeCheckPrompt, quizeResultPrompt: quizeResultPrompt}
}

// GetPersonalizedQuestions returns the city-mode quiz for city, accepted
// in any casing/spacing. The first request for a city has the LLM generate
// its questions and caches them in the database; every later request is
// served from that cache, in quiz order.
//
// An unknown city returns InvalidCityError and no questions. A cache
// lookup, generation, or caching failure returns the underlying error
// unchanged — when caching fails the generated questions are returned
// alongside the error, and some of them may already be persisted.
func (s *quizService) GetPersonalizedQuestions(ctx context.Context, city string) (q []Question, err error) {
	log := logging.From(ctx)
	log.Debug("quizService.GetPersonalizedQuestions", zap.String("city", city))

	label, ok := s.validator.GetCityID(city)
	if !ok {
		err = InvalidCityError
		log.Error("unknown city requested", zap.String("city", city))
		return
	}

	q, err = s.getStoredQuestions(ctx, label)
	if err != nil {
		return
	}

	if len(q) > 0 {
		log.Info("serving cached city quiz", zap.String("city", label), zap.Int("questions", len(q)))
		return
	}

	q, err = s.generateQuestions(ctx, label)
	if err != nil {
		return
	}

	err = s.storeQuestions(ctx, label, q)
	if err != nil {
		return
	}

	log.Info("city quiz generated and cached", zap.String("city", label), zap.Int("questions", len(q)))
	return
}

// GetPreStoredQuestions returns the fixed match-mode question bank in quiz
// order. An unseeded bank yields an empty slice with a nil error; a
// database failure returns that error unchanged and no questions.
func (s *quizService) GetPreStoredQuestions(ctx context.Context) (arr []Question, err error) {
	log := logging.From(ctx)
	log.Debug("quizService.GetPreStoredQuestions")

	storedQuestions, err := s.sql.GetMatchQuestions(ctx)
	if err != nil {
		log.Error("loading match question bank", zap.Error(err))
		return
	}

	arr = make([]Question, 0, len(storedQuestions))

	for _, q := range storedQuestions {
		arr = append(arr, Question{
			ID:      q.MeaningID,
			Text:    q.Text,
			Options: q.Options,
		})
	}

	return
}

// GetResults resolves a finished quiz into a stored, shareable result.
// The submission (mode, city, and the answers in order) is canonicalized
// into a JSON combination that acts as the deduplication key: a
// combination seen before returns its previously stored result without
// touching the LLM, a new one has the LLM generate the verdict exactly
// once, stores it under a fresh permanent id, and returns that. In city
// mode the verdict's city is forced to the submitted city; in match mode
// the LLM's pick stands. Concurrent identical submissions are safe — both
// get the same stored row.
//
// The returned QuizResult always carries the permanent id it can later be
// fetched by. On any failure — marshaling, database, the LLM call, or an
// LLM reply that isn't valid result JSON — the error comes back unchanged
// and the result is zero-valued; a failure after generation but during
// save means the verdict is lost and the next identical submission
// regenerates it.
func (s *quizService) GetResults(ctx context.Context, mode, city string, answers []Answer) (results types.QuizResult, err error) {
	log := logging.From(ctx)
	log.Debug("quizService.GetResults", zap.String("mode", mode), zap.String("city", city), zap.Any("answers", answers))

	combination, err := sonic.Marshal(struct {
		Mode    string   `json:"mode"`
		City    string   `json:"city"`
		Answers []Answer `json:"answers"`
	}{
		Mode:    mode,
		City:    city,
		Answers: answers,
	})
	if err != nil {
		log.Error("canonicalizing submission", zap.Error(err), zap.String("mode", mode), zap.String("city", city))
		return
	}

	stored, err := s.sql.GetResultByCombination(ctx, string(combination))
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Error("looking up stored result", zap.Error(err))
	}
	if errors.Is(err, pgx.ErrNoRows) {
		log.Info("new submission — generating verdict", zap.String("mode", mode), zap.String("city", city))

		var generated types.QuizResult
		generated, err = s.generateResult(ctx, mode, city, answers)
		if err != nil {
			return
		}

		// in city mode the verdict is about the requested city; in match
		// mode the LLM picks the city
		if mode == "city" {
			generated.City = city
		}

		stored, err = s.sql.SaveResult(ctx, SaveResultParams{
			Combination:  string(combination),
			Mode:         mode,
			City:         generated.City,
			Score:        int32(generated.Score),
			Title:        generated.Title,
			Summary:      generated.Summary,
			GreenFlags:   generated.GreenFlags,
			RedFlags:     generated.RedFlags,
			Alternatives: generated.Alternatives,
			Closing:      generated.Closing,
		})
		if err != nil {
			log.Error("saving generated verdict", zap.Error(err), zap.String("city", generated.City))
		} else {
			log.Info("verdict stored", zap.String("id", stored.ID), zap.String("city", stored.City))
		}
	} else if err == nil {
		log.Info("reusing stored verdict", zap.String("id", stored.ID))
	}
	if err != nil {
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

// getStoredQuestions loads the cached quiz for one city, in quiz order.
// label must be the canonical city label from the validator — any other
// spelling simply finds nothing. A city with no cached questions yields an
// empty slice and a nil error; a database failure returns the error
// unchanged.
func (s *quizService) getStoredQuestions(ctx context.Context, label string) (q []Question, err error) {
	log := logging.From(ctx)
	log.Debug("quizService.getStoredQuestions", zap.String("label", label))

	questions, err := s.sql.GetCityQuestions(ctx, label)
	if err != nil {
		log.Error("loading cached city questions", zap.Error(err), zap.String("label", label))
		return
	}

	for _, qs := range questions {
		q = append(q, Question{
			ID:      qs.MeaningID,
			Text:    qs.Text,
			Options: qs.Options,
		})
	}

	return
}

// storeQuestions writes a freshly generated quiz into the cache, recording
// slice order as quiz order. label must be the canonical label of a city
// present in the cities table, otherwise the insert fails its foreign key;
// the cache is also write-once, so re-storing an existing (city, question)
// pair fails on the primary key. Returns nil only when every row landed.
// Inserts are not transactional: on error, the rows before the failing one
// stay written.
func (s *quizService) storeQuestions(ctx context.Context, label string, questions []Question) error {
	log := logging.From(ctx)
	log.Debug("quizService.storeQuestions", zap.String("label", label), zap.Any("questions", questions))

	for i, q := range questions {
		err := s.sql.AddCityQuestion(ctx, AddCityQuestionParams{
			CityLabel: label,
			Position:  int32(i),
			MeaningID: q.ID,
			Text:      q.Text,
			Options:   q.Options,
		})
		if err != nil {
			log.Error("caching city question", zap.Error(err),
				zap.String("label", label), zap.String("meaning_id", q.ID), zap.Int("position", i))
			return err
		}
	}

	return nil
}

// generateQuestions asks the LLM to write the quiz for the city named by
// label and parses the reply. It returns the parsed questions, the API
// error when the call fails, or an unmarshal error when the reply isn't
// the expected JSON array.
func (s *quizService) generateQuestions(ctx context.Context, label string) (q []Question, err error) {
	log := logging.From(ctx)
	log.Debug("quizService.generateQuestions", zap.String("label", label))

	chatCompletion, err := s.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.DeveloperMessage(s.lowkeyCityVibeCheckPrompt),
			openai.UserMessage("City under lowkey vibe check: " + label),
		},
		Model: openai.ChatModelGPT5_2,
	})
	if err != nil {
		log.Error("LLM question generation failed", zap.Error(err), zap.String("label", label))
		return
	}

	err = sonic.Unmarshal([]byte(chatCompletion.Choices[0].Message.Content), &q)
	if err != nil {
		log.Error("LLM reply is not a question array", zap.Error(err), zap.String("label", label))
	}
	return
}

// generateResult asks the LLM for the verdict on a finished quiz described
// by mode, city and answers, and parses the reply. It returns the parsed
// QuizResult (id-less — storage assigns the id), the API error when the
// call fails, or an unmarshal error when the reply isn't the expected JSON.
func (s *quizService) generateResult(ctx context.Context, mode, city string, answers []Answer) (result types.QuizResult, err error) {
	log := logging.From(ctx)
	log.Debug("quizService.generateResult", zap.String("mode", mode), zap.String("city", city), zap.Any("answers", answers))

	chatCompletion, err := s.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.DeveloperMessage(s.quizeResultPrompt),
			openai.UserMessage("City under lowkey vibe check: " + city + "\n" + "Answers: " + fmt.Sprintf("%v", answers)),
		},
		Model: openai.ChatModelGPT5_2,
	})
	if err != nil {
		log.Error("LLM verdict generation failed", zap.Error(err), zap.String("mode", mode), zap.String("city", city))
		return
	}

	err = sonic.Unmarshal([]byte(chatCompletion.Choices[0].Message.Content), &result)
	if err != nil {
		log.Error("LLM reply is not a result", zap.Error(err), zap.String("mode", mode), zap.String("city", city))
	}
	return
}
