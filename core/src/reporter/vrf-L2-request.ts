import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import {
  L2_REPORTER_VRF_REQUEST_QUEUE_NAME,
  L2_VRF_REQUEST_REPORTER_STATE_NAME,
  L2_VRF_REQUEST_SERVICE_NAME,
  NONCE_MANAGER_L2_VRF_REQUEST_QUEUE_NAME
} from '../settings'
import { factory } from './factory'

export async function buildReporter(redisClient: RedisClientType, logger: Logger) {
  await factory({
    redisClient,
    stateName: L2_VRF_REQUEST_REPORTER_STATE_NAME,
    nonceManagerQueueName: NONCE_MANAGER_L2_VRF_REQUEST_QUEUE_NAME,
    service: L2_VRF_REQUEST_SERVICE_NAME,
    reporterQueueName: L2_REPORTER_VRF_REQUEST_QUEUE_NAME,
    concurrency: 1,
    delegatedFee: false,
    _logger: logger
  })
}
