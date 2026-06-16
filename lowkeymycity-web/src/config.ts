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
