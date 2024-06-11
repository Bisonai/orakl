import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import {
  CONCURRENCY,
  REPORTER_VRF_QUEUE_NAME,
  VRF_REPORTER_STATE_NAME,
  VRF_SERVICE_NAME
} from '../settings'
import { factory } from './factory'

export async function buildReporter(redisClient: RedisClientType, logger: Logger) {
  await factory({
    redisClient,
    stateName: VRF_REPORTER_STATE_NAME,
    service: VRF_SERVICE_NAME,
    reporterQueueName: REPORTER_VRF_QUEUE_NAME,
    concurrency: CONCURRENCY,
    delegatedFee: false,
    _logger: logger
  })
}
