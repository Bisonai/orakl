import * as dotenv from 'dotenv'
import { ethers } from 'ethers'
dotenv.config()

export const FETCHER_QUEUE_NAME = 'orakl-fetcher-queue'

export const WORKER_OPTS = { concurrency: Number(process.env.CONCURRENCY) || 20 }

export const FETCH_FREQUENCY = 2_000

export const DEVIATION_QUEUE_NAME = 'orakl-deviation-queue'

export const FETCHER_TYPE = process.env.FETCHER_TYPE || 0

export const PROVIDER_URL = process.env.PROVIDER_URL || 'http://127.0.0.1:8545'

function createJsonRpcProvider(providerUrl: string = PROVIDER_URL) {
  return new ethers.JsonRpcProvider(providerUrl)
}

export const PROVIDER = createJsonRpcProvider()
