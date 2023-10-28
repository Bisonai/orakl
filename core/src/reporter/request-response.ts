import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import {
  REPORTER_REQUEST_RESPONSE_QUEUE_NAME,
  REQUEST_RESPONSE_REPORTER_STATE_NAME,
  REQUEST_RESPONSE_SERVICE_NAME
} from '../settings'
import { factory } from './factory'

export async function buildReporter(redisClient: RedisClientType, logger: Logger) {
  await factory({
    redisClient,
    stateName: REQUEST_RESPONSE_REPORTER_STATE_NAME,
    service: REQUEST_RESPONSE_SERVICE_NAME,
    reporterQueueName: REPORTER_REQUEST_RESPONSE_QUEUE_NAME,
    concurrency: 1,
    delegatedFee: false,
    _logger: logger
  })
}
