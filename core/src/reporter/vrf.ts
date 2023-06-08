import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { factory } from './factory'
import { REPORTER_VRF_QUEUE_NAME, VRF_REPORTER_STATE_NAME, VRF_SERVICE_NAME } from '../settings'

export async function buildReporter(redisClient: RedisClientType, logger: Logger) {
  await factory({
    redisClient,
    stateName: VRF_REPORTER_STATE_NAME,
    service: VRF_SERVICE_NAME,
    reporterQueueName: REPORTER_VRF_QUEUE_NAME,
    concurrency: 1,
    delegatedFee: false,
    _logger: logger
  })
}
