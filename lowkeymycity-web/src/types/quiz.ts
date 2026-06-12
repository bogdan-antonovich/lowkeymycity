export type QuizMode = 'city' | 'match'

// One quiz question. `id` is the vibe axis it measures (climate, pace,
// social battery, ...) — stable across modes.
export interface QuizQuestion {
  id: string
  text: string
  options: string[]
}

export interface QuizAnswer {
  questionId: string
  question: string
  answer: string
}

// What the frontend POSTs to /api/quiz/result.
export interface QuizSubmission {
  mode: QuizMode
  city?: string
  answers: QuizAnswer[]
}

export interface Alternative {
  city: string
  blurb: string
}

// Shape of what the backend returns after a quiz. `score` only exists for
// mode 'city'. `greenFlags` and `redFlags` are arrays of paragraphs. For
// mode 'city' the alternatives are "better matches", for 'match' they're
// the runner-up cities.
export interface QuizResult {
  // present when the result came from (and is stored by) the backend
  id?: string
  mode: QuizMode
  city: string
  title: string
  summary: string
  greenFlags: string[]
  redFlags: string[]
  alternatives: Alternative[]
  closing: string
  score?: number
}
