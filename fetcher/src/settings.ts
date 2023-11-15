import * as dotenv from 'dotenv'
dotenv.config()

export const FETCHER_QUEUE_NAME = 'orakl-fetcher-queue'

export const WORKER_OPTS = { concurrency: process.env.CONCURRENCY || 20 }

export const FETCH_FREQUENCY = 2_000

export const DEVIATION_QUEUE_NAME = 'orakl-deviation-queue'

export const FETCHER_TYPE = process.env.FETCHER_TYPE || 0
