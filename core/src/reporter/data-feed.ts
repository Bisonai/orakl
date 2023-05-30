import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { factory } from './factory'
import {
  REPORTER_AGGREGATOR_QUEUE_NAME,
  DATA_FEED_REPORTER_STATE_NAME,
  DATA_FEED_SERVICE_NAME
} from '../settings'

export async function buildReporter(redisClient: RedisClientType, logger: Logger) {
  await factory({
    redisClient,
    stateName: DATA_FEED_REPORTER_STATE_NAME,
    service: DATA_FEED_SERVICE_NAME,
    reporterQueueName: REPORTER_AGGREGATOR_QUEUE_NAME,
    concurrency: 5,
    delegatedFee: true,
    _logger: logger
  })
}
