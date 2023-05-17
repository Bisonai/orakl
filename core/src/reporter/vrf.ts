import { Worker } from 'bullmq'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { loadWalletParameters, buildWallet } from './utils'
import { REPORTER_VRF_QUEUE_NAME, BULLMQ_CONNECTION } from '../settings'
import { job } from './job'

const FILE_NAME = import.meta.url

export async function reporter(redisClient: RedisClientType, logger: Logger) {
  const _logger = logger.child({ name: 'reporter', file: FILE_NAME })
  const { privateKey, providerUrl } = loadWalletParameters()
  const wallet = await buildWallet({ privateKey, providerUrl })
  new Worker(REPORTER_VRF_QUEUE_NAME, await job(wallet, _logger), BULLMQ_CONNECTION)
}
