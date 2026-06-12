// Swap in the real Buy Me a Coffee username when it exists.
export const BUY_ME_A_COFFEE_URL = 'https://buymeacoffee.com/lowkeymycity'

// Where the api publishes the exported city list. In dev this resolves to
// web/public/cities.json so everything works without the backend; in
// production the validator-exported file is served by the api.
export const CITIES_URL = import.meta.env.DEV ? '/cities.json' : '/api/v1/cities'
