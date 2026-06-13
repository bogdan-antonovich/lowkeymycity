package quiz

import (
	"errors"
	"lowkeymycity/pkg/logging"
	_ "lowkeymycity/pkg/types"
	"net/http"

	"github.com/labstack/echo/v5"
	"go.uber.org/zap"
)

// Controller registers the quiz HTTP routes on the versioned API group.
type Controller interface {
	GetQuestions(g *echo.Group)
	GetResults(g *echo.Group)
}

// quizController implements Controller; its handlers do binding and JSON
// shaping only and delegate everything else to qs.
type quizController struct {
	qs QuizService
}

// Answer is one answered question as submitted by the frontend. The full
// question and answer text travel with the id because the verdict is
// generated from them as-is — the backend never looks the question back up.
type Answer struct {
	QuestionID string `json:"questionId" validate:"required"`
	Question   string `json:"question" validate:"required"`
	Answer     string `json:"answer" validate:"required"`
}

// GetResultsRequest is the POST /quiz/result body. City is required in
// city mode and must name a known US city whenever present; match mode
// may omit it.
type GetResultsRequest struct {
	Mode    string   `json:"mode" validate:"required,oneof=match city"`
	City    string   `json:"city" validate:"required_if=Mode city,omitempty,city_exists"`
	Answers []Answer `json:"answers" validate:"required,min=1,dive"`
}

// GetQuestionsRequest holds the GET /quiz parameters, bound from the query
// string. Mode is strictly match or city; city is required in city mode
// and must name a known US city whenever present.
type GetQuestionsRequest struct {
	Mode string `json:"mode" query:"mode" validate:"required,oneof=match city"`
	City string `json:"city" query:"city" validate:"required_if=Mode city,omitempty,city_exists"`
}

// QuestionResponse is one quiz question as served over the wire. Options
// are plain text strings — the {id,text} shape used internally is not
// exposed to the frontend.
type QuestionResponse struct {
	ID      string   `json:"id"`
	Text    string   `json:"text"`
	Options []string `json:"options"`
}

// GetQuestionsResponse is the GET /quiz payload: the question set plus the
// city it was asked for, echoed back as submitted — omitted when absent,
// as in match mode. The shape mirrors what the frontend expects (and what
// its mock path builds).
type GetQuestionsResponse struct {
	City      string             `json:"city,omitempty"`
	Questions []QuestionResponse `json:"questions"`
}

func toQuestionResponses(questions []Question) []QuestionResponse {
	out := make([]QuestionResponse, len(questions))
	for i, q := range questions {
		opts := make([]string, len(q.Options))
		for j, o := range q.Options {
			opts[j] = o.Text
		}
		out[i] = QuestionResponse{ID: q.ID, Text: q.Text, Options: opts}
	}
	return out
}

// NewQuizController builds the quiz controller around qs, which must be a
// non-nil QuizService — nil isn't rejected here but panics on the first
// request. The returned Controller is inert until its route-registering
// methods are called.
func NewQuizController(qs QuizService) Controller {
	return &quizController{qs: qs}
}

// GetQuestions registers GET /quiz on g (mounted under the API prefix), the endpoint serving the question
// set for a quiz run from its bound mode and city parameters.
//
// Mode "match" answers 200 with the fixed match-mode question bank,
// wrapped as GetQuestionsResponse. Mode "city" resolves city against the
// known-US-cities set and answers its cached (or first-time LLM-generated,
// then cached) questions the same way, the requested city echoed
// alongside. Anything else — a missing or unrecognized mode, a missing or
// unknown city — fails validation with a 400 before any work happens; a
// database/LLM failure surfaces through Echo's error handler as a 500.
//
// @Summary      Quiz questions
// @Description  Question set for a quiz run: fixed bank for match mode, LLM-generated cached questions for city mode.
// @Tags         quiz
// @Produce      json
// @Param        mode  query  string  true   "quiz mode, strictly one of the enum"  Enums(match, city)
// @Param        city  query  string  false  "known US city name; required when mode=city"
// @Success      200  {object}  quiz.GetQuestionsResponse
// @Failure      400  {object}  echo.HTTPError
// @Failure      500  {object}  echo.HTTPError
// @Router       /quiz [get]
func (qc *quizController) GetQuestions(g *echo.Group) {
	g.GET("/quiz", func(c *echo.Context) error {
		log := logging.From(c.Request().Context())

		var req GetQuestionsRequest
		if err := c.Bind(&req); err != nil {
			log.Error("binding GET /quiz request", zap.Error(err))
			return err
		}
		log.Debug("GET /quiz", zap.String("mode", req.Mode), zap.String("city", req.City))

		// the validator shaped the 400; the failure's one Error line is
		// written here, where the request id is in reach
		if err := c.Validate(&req); err != nil {
			log.Error("request validation failed", zap.Error(err))
			return err
		}

		if req.Mode == "match" {

			questions, err := qc.qs.GetPreStoredQuestions(c.Request().Context())
			if err != nil {
				return err
			}

			return c.JSON(http.StatusOK, GetQuestionsResponse{City: req.City, Questions: toQuestionResponses(questions)})

		} else {

			questions, err := qc.qs.GetPersonalizedQuestions(c.Request().Context(), req.City)
			if errors.Is(err, ErrInvalidCity) {
				return echo.NewHTTPError(http.StatusBadRequest, "unknown city")
			}
			if err != nil {
				return err
			}

			return c.JSON(http.StatusOK, GetQuestionsResponse{City: req.City, Questions: toQuestionResponses(questions)})
		}
	})
}

// GetResults registers POST /quiz/result on g (mounted under the API prefix), the endpoint that turns a
// finished quiz into a stored, shareable result.
//
// The JSON body (GetResultsRequest: mode, city, answers) is resolved by
// the quiz service, which reuses the stored verdict when this exact
// submission was seen before and asks the LLM exactly once when it wasn't.
// Success is a 200 with the stored QuizResult, permanent id included — the
// id the frontend builds /r/{id} and the PDF link from. An unparseable
// body — or one that fails validation: bad mode, missing or unknown city
// in city mode, no answers — is a 400; database or LLM failures surface
// as 500s.
//
// @Summary      Submit quiz answers
// @Description  Stores (or reuses) the verdict for an answer combination and returns it with its permanent id.
// @Tags         quiz
// @Accept       json
// @Produce      json
// @Param        request  body  quiz.GetResultsRequest  true  "quiz answers"
// @Success      200  {object}  types.QuizResult
// @Failure      400  {object}  echo.HTTPError
// @Failure      500  {object}  echo.HTTPError
// @Router       /quiz/result [post]
func (qc *quizController) GetResults(g *echo.Group) {
	g.POST("/quiz/result", func(c *echo.Context) error {
		log := logging.From(c.Request().Context())

		var req GetResultsRequest
		if err := c.Bind(&req); err != nil {
			log.Error("binding POST /quiz/result request", zap.Error(err))
			return err
		}
		log.Debug("POST /quiz/result",
			zap.String("mode", req.Mode), zap.String("city", req.City), zap.Any("answers", req.Answers))

		// the validator shaped the 400; the failure's one Error line is
		// written here, where the request id is in reach
		if err := c.Validate(&req); err != nil {
			log.Error("request validation failed", zap.Error(err))
			return err
		}

		results, err := qc.qs.GetResults(c.Request().Context(), req.Mode, req.City, req.Answers)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, results)
	})
}
