import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { factory } from './factory'
import {
  DATA_FEED_L2_SERVICE_NAME,
  DATA_FEED_REPORTER_L2_STATE_NAME,
  REPORTER_AGGREGATOR_L2_QUEUE_NAME,
  L2_PROVIDER_URL,
  L2_CHAIN
} from '../settings'

export async function buildReporter(redisClient: RedisClientType, logger: Logger) {
  await factory({
    redisClient,
    stateName: DATA_FEED_REPORTER_L2_STATE_NAME,
    service: DATA_FEED_L2_SERVICE_NAME,
    reporterQueueName: REPORTER_AGGREGATOR_L2_QUEUE_NAME,
    concurrency: 5,
    delegatedFee: false,
    _logger: logger,
    providerUrl: L2_PROVIDER_URL,
    chain: L2_CHAIN
  })
}
