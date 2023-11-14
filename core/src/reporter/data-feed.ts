import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import {
  BAOBAB_CHAIN_ID,
  CYPRESS_CHAIN_ID,
  DATA_FEED_REPORTER_STATE_NAME,
  DATA_FEED_SERVICE_NAME,
  PROVIDER,
  REPORTER_AGGREGATOR_QUEUE_NAME
} from '../settings'
import { factory } from './factory'

export async function buildReporter(redisClient: RedisClientType, logger: Logger) {
  const chainId = (await PROVIDER.getNetwork()).chainId

  await factory({
    redisClient,
    stateName: DATA_FEED_REPORTER_STATE_NAME,
    service: DATA_FEED_SERVICE_NAME,
    reporterQueueName: REPORTER_AGGREGATOR_QUEUE_NAME,
    concurrency: 10,
    delegatedFee: [BAOBAB_CHAIN_ID, CYPRESS_CHAIN_ID].includes(chainId) ? true : false,
    _logger: logger
  })
}
