import ethers from 'ethers'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import {
  DATA_FEED_REPORTER_STATE_NAME,
  DATA_FEED_SERVICE_NAME,
  PROVIDER_URL,
  REPORTER_AGGREGATOR_QUEUE_NAME
} from '../settings'
import { factory } from './factory'

export async function buildReporter(redisClient: RedisClientType, logger: Logger) {
  const provider = new ethers.providers.JsonRpcProvider(PROVIDER_URL)
  const chainId = (await provider.getNetwork()).chainId

  await factory({
    redisClient,
    stateName: DATA_FEED_REPORTER_STATE_NAME,
    service: DATA_FEED_SERVICE_NAME,
    reporterQueueName: REPORTER_AGGREGATOR_QUEUE_NAME,
    concurrency: 10,
    delegatedFee: [1001, 8217].includes(chainId) ? true : false,
    _logger: logger
  })
}
