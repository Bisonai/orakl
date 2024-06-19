import * as dotenv from 'dotenv'
dotenv.config()

export const FETCHER_QUEUE_NAME = 'orakl-fetcher-queue'

export const WORKER_OPTS = { concurrency: Number(process.env.CONCURRENCY) || 20 }

export const FETCH_FREQUENCY = 2_000

export const FETCH_TIMEOUT = 1_000

export const DEVIATION_QUEUE_NAME = 'orakl-deviation-queue'

export const FETCHER_TYPE = process.env.FETCHER_TYPE || 0

export const CYPRESS_PROVIDER_URL =
  process.env.CYPRESS_PROVIDER_URL || 'https://public-en-cypress.klaytn.net'

export const ETHEREUM_PROVIDER_URL =
  process.env.ETHEREUM_PROVIDER_URL || 'https://ethereum-mainnet.g.allthatnode.com/full/evm'
