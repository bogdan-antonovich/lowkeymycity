package types

import "go.uber.org/zap"

// Option is one selectable answer of a quiz question.
type Option struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// Alternative is one runner-up city suggestion inside a result.
type Alternative struct {
	City  string `json:"city"`
	Blurb string `json:"blurb"`
}

// QuizResult is the stored verdict for one quiz run, shaped exactly like
// QuizResult in web/src/types/quiz.ts. Score is meaningful only in city
// mode and is omitted from JSON when zero.
type QuizResult struct {
	ID           string        `json:"id"`
	Mode         string        `json:"mode"`
	City         string        `json:"city"`
	Score        int           `json:"score,omitempty"`
	Title        string        `json:"title"`
	Summary      string        `json:"summary"`
	GreenFlags   []string      `json:"greenFlags"`
	RedFlags     []string      `json:"redFlags"`
	Alternatives []Alternative `json:"alternatives"`
	Closing      string        `json:"closing"`
}

type Logger interface {
	Debug(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
}
