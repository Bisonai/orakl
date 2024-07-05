import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import {
  L2_CHAIN,
  L2_DATA_FEED_REPORTER_STATE_NAME,
  L2_DATA_FEED_SERVICE_NAME,
  L2_PROVIDER_URL,
  L2_REPORTER_AGGREGATOR_QUEUE_NAME,
} from '../settings'
import { factory } from './factory'

export async function buildReporter(redisClient: RedisClientType, logger: Logger) {
  await factory({
    redisClient,
    stateName: L2_DATA_FEED_REPORTER_STATE_NAME,
    nonceManagerQueueName: 'NONCE_MANAGER_L2_DATA_FEED_QUEUE_NAME',
    service: L2_DATA_FEED_SERVICE_NAME,
    reporterQueueName: L2_REPORTER_AGGREGATOR_QUEUE_NAME,
    concurrency: 5,
    delegatedFee: false,
    _logger: logger,
    providerUrl: L2_PROVIDER_URL,
    chain: L2_CHAIN,
  })
}
