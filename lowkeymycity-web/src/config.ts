// Canonical origin used for absolute URLs in canonical links, Open Graph
// tags, and the sitemap. If the site ever moves, change it here only.
export const SITE_URL = 'https://lowkeymycity.com'
export const SITE_NAME = 'lowkeymycity'

// Default social/SEO copy, used on the home page and as the fallback for
// any route that doesn't set its own. Lowercase, conversational, same
// voice as the on-page copy.
//
// DEFAULT_TITLE is the document/tab title (brand-prefixed for search).
// SOCIAL_TITLE is the og/twitter headline: just the tagline, since the
// og:site_name already carries the brand and repeating it reads clumsy.
export const DEFAULT_TITLE = 'lowkeymycity: how lowkey is your city, actually?'
export const SOCIAL_TITLE = 'how lowkey is your city, actually?'
export const DEFAULT_DESCRIPTION =
  "twelve questions, about 90 seconds, one honest verdict on how lowkey your city actually is. no city in mind? we'll name one that matches your energy."

// 1200×630 social share card served from web/public/og.png.
export const OG_IMAGE = `${SITE_URL}/og.png`

export interface CryptoWallet {
  /** Display name shown in the dialog, e.g. 'Bitcoin'. */
  name: string
  /** Short ticker shown in the chip, e.g. 'BTC'. */
  symbol: string
  /** Network label so donors send on the right chain, e.g. 'Bitcoin', 'ERC-20', 'Solana', 'TRC-20'. */
  network: string
  /** The receiving address. Paste the real one here. Placeholders are obviously fake on purpose. */
  address: string
}

export const CRYPTO_WALLETS: CryptoWallet[] = [
  {
    name: 'Bitcoin',
    symbol: 'BTC',
    network: 'Bitcoin',
    address: '1LdrpHv4h5UoCtxNfqL5PJVAhmyMtvfthv',
  },
  {
    name: 'Ethereum',
    symbol: 'ETH',
    network: 'ERC-20',
    address: '0x434a943fb69096e320131cbc7e3b2f7aa1fcc3a1',
  },
  {
    name: 'Solana',
    symbol: 'SOL',
    network: 'Solana',
    address: '6di6CHUqVSo3TRugT2GiK5iSbfvYojynVbULBomArAAx',
  },
  {
    name: 'Tether',
    symbol: 'USDT',
    network: 'TRC-20 (Tron)',
    address: 'TSadiY9AjzQ36HDy1UcJ9aj3G7c97PgDr6',
  },
]

// Where the api publishes the exported city list. In dev this resolves to
// web/public/cities.json so everything works without the backend; in
// production the validator-exported file is served by the api.
export const CITIES_URL = import.meta.env.DEV ? '/cities.json' : '/api/v1/cities'
