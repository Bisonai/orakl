import { Queue } from 'bullmq'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import {
  BAOBAB_CHAIN_ID,
  BULLMQ_CONNECTION,
  CYPRESS_CHAIN_ID,
  DATA_FEED_REPORTER_CONCURRENCY,
  DATA_FEED_REPORTER_STATE_NAME,
  DATA_FEED_SERVICE_NAME,
  PROVIDER,
  REPORTER_AGGREGATOR_QUEUE_NAME
} from '../settings'
import { factory } from './factory'

export async function buildReporter(redisClient: RedisClientType, logger: Logger) {
  const chainId = (await PROVIDER.getNetwork()).chainId

  const reporterAggregateQueue = new Queue(REPORTER_AGGREGATOR_QUEUE_NAME, BULLMQ_CONNECTION)
  await reporterAggregateQueue.obliterate({ force: true })

  await factory({
    redisClient,
    stateName: DATA_FEED_REPORTER_STATE_NAME,
    nonceManagerQueueName: 'NONCE_MANAGER_DATA_FEED_QUEUE_NAME',
    service: DATA_FEED_SERVICE_NAME,
    reporterQueueName: REPORTER_AGGREGATOR_QUEUE_NAME,
    concurrency: DATA_FEED_REPORTER_CONCURRENCY,
    delegatedFee: [BAOBAB_CHAIN_ID, CYPRESS_CHAIN_ID].includes(chainId) ? true : false,
    _logger: logger
  })
}
