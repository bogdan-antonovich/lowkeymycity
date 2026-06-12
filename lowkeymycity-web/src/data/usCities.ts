import { CITIES_URL } from '@/config'

// The city list is produced by the api (a Go job exports the `cities` table,
// ordered by population descending, to a JSON file served over http). The
// dev server serves web/public/cities.json at the same URL so everything
// works without the backend. Population order matters: the autocomplete
// ranks suggestions by array position.
let cities: string[] | null = null
let pending: Promise<string[]> | null = null

export function loadUsCities(): Promise<string[]> {
  if (cities) return Promise.resolve(cities)
  pending ??= fetch(CITIES_URL)
    .then((response) => {
      if (!response.ok) throw new Error(`cities fetch failed: ${response.status}`)
      return response.json() as Promise<string[]>
    })
    .then((list) => (cities = list))
    .catch((error) => {
      pending = null // allow a retry on the next call
      throw error
    })
  return pending
}
