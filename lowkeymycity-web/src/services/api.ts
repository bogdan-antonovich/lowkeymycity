import { cityQuestions, MATCH_QUESTIONS } from '@/data/mockQuestions'
import { MOCK_RESULTS } from '@/data/mockResults'
import type { QuizMode, QuizQuestion, QuizResult, QuizSubmission } from '@/types/quiz'

// Flip to the real backend with VITE_USE_MOCKS=false (see vite proxy config).
export const USE_MOCKS = import.meta.env.VITE_USE_MOCKS !== 'false'

export interface QuestionsResponse {
  city?: string
  questions: QuizQuestion[]
}

const MOCK_BY_ID: Record<string, QuizResult> = {
  'demo-city-high': MOCK_RESULTS.cityHigh,
  'demo-city-low': MOCK_RESULTS.cityLow,
  'demo-match': MOCK_RESULTS.match,
}

// Results "stored" by mock submissions. Lives only until refresh, which
// conveniently also demos the result-not-found page.
const mockStore = new Map<string, QuizResult>()

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))

export async function getQuestions(mode: QuizMode, city?: string): Promise<QuestionsResponse> {
  if (USE_MOCKS) {
    // city questions are LLM-generated on the real backend, so they take longer
    await sleep(mode === 'city' ? 1800 : 700)
    return {
      city,
      questions: mode === 'city' && city ? cityQuestions(city) : MATCH_QUESTIONS,
    }
  }

  const params = new URLSearchParams({ mode })
  if (city) params.set('city', city)
  const response = await fetch(`/api/v1/quiz?${params}`)
  if (!response.ok) throw new Error(`request failed: ${response.status}`)
  return response.json()
}

export async function submitQuiz(submission: QuizSubmission): Promise<QuizResult> {
  if (USE_MOCKS) {
    await sleep(3000)
    const id = `demo-${Date.now().toString(36)}`
    let result: QuizResult
    if (submission.mode === 'match') {
      result = { ...MOCK_RESULTS.match, id }
    } else {
      const lucky = Math.random() < 0.5
      const template = lucky ? MOCK_RESULTS.cityHigh : MOCK_RESULTS.cityLow
      const city = submission.city ?? template.city
      const name = city.split(',')[0].toLowerCase()
      result = {
        ...template,
        id,
        city,
        title: lucky ? `${name} is lowkey your soulmate` : `${name} said you're too soft for this`,
      }
    }
    mockStore.set(id, result)
    return result
  }

  const response = await fetch('/api/v1/quiz/result', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(submission),
  })
  if (!response.ok) throw new Error(`request failed: ${response.status}`)
  return response.json()
}

export async function getResult(id: string): Promise<QuizResult> {
  if (USE_MOCKS) {
    await sleep(600)
    const result = mockStore.get(id) ?? MOCK_BY_ID[id]
    if (!result) throw new Error('result not found')
    return { ...result, id }
  }

  const response = await fetch(`/api/v1/results/${encodeURIComponent(id)}`)
  if (!response.ok) throw new Error(`request failed: ${response.status}`)
  return response.json()
}

export function resultPdfUrl(id: string): string {
  return `/api/v1/results/${encodeURIComponent(id)}/pdf`
}
