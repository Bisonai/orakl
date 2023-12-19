import { Worker } from 'bullmq'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { BULLMQ_CONNECTION, CHAIN, PROVIDER_URL } from '../settings'
import { reporter } from './reporter'
import { State } from './state'
import { workerType } from './types'
import { watchman } from './watchman'

const FILE_NAME = import.meta.url

export async function factory({
  redisClient,
  stateName,
  service,
  reporterQueueName,
  concurrency,
  delegatedFee,
  _logger,
  providerUrl = PROVIDER_URL,
  chain = CHAIN,
  _worker = reporter
}: {
  redisClient: RedisClientType
  stateName: string
  service: string
  reporterQueueName: string
  concurrency: number
  delegatedFee: boolean
  providerUrl?: string
  chain?: string
  _logger: Logger
  _worker?: workerType
}) {
  const logger = _logger.child({ name: 'reporter', file: FILE_NAME })

  const state = new State({
    redisClient,
    providerUrl,
    stateName,
    service,
    chain,
    delegatedFee,
    logger
  })
  await state.refresh()

  logger.debug(await state.active(), 'Active reporters')

  const reporterWorker = new Worker(reporterQueueName, await _worker(state, logger), {
    ...BULLMQ_CONNECTION,
    concurrency
  })
  reporterWorker.on('error', (e) => {
    logger.error(e)
  })

  const watchmanServer = await watchman({ state, logger })

  async function handleExit() {
    logger.info('Exiting. Wait for graceful shutdown.')

    await redisClient.quit()
    await reporterWorker.close()
    await watchmanServer.close()
  }
  process.on('SIGINT', handleExit)
  process.on('SIGTERM', handleExit)
}
