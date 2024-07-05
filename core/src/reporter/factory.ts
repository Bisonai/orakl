import { Queue, Worker } from 'bullmq'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { BULLMQ_CONNECTION, CHAIN, PROVIDER_URL } from '../settings'
import { nonceManager } from './nonceManager'
import { reporter } from './reporter'
import { State } from './state'
import { watchman } from './watchman'

const FILE_NAME = import.meta.url

export async function factory({
  redisClient,
  stateName,
  service,
  nonceManagerQueueName,
  reporterQueueName,
  concurrency,
  delegatedFee,
  _logger,
  providerUrl = PROVIDER_URL,
  chain = CHAIN,
}: {
  redisClient: RedisClientType
  stateName: string
  service: string
  nonceManagerQueueName: string
  reporterQueueName: string
  concurrency: number
  delegatedFee: boolean
  providerUrl?: string
  chain?: string
  _logger: Logger
}) {
  const logger = _logger.child({ name: 'reporter', file: FILE_NAME })

  const state = new State({
    redisClient,
    providerUrl,
    stateName,
    service,
    chain,
    delegatedFee,
    logger,
  })
  await state.refresh()

  const activeReporters = await state.active()
  logger.debug(
    activeReporters.map((x) => {
      return { address: x.address, oracleAddress: x.oracleAddress }
    }),
    'Active reporters',
  )

  const reporterQueue = new Queue(reporterQueueName, BULLMQ_CONNECTION)
  const nonceManagerWorker = new Worker(
    nonceManagerQueueName,
    await nonceManager(reporterQueue, service, state, logger),
    {
      ...BULLMQ_CONNECTION,
      concurrency,
    },
  )
  nonceManagerWorker.on('error', (e) => {
    logger.error(e)
  })

  const reporterWorker = new Worker(reporterQueueName, await reporter(state, logger), {
    ...BULLMQ_CONNECTION,
    concurrency,
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
