import { Job, Worker, Queue } from 'bullmq'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { Aggregator__factory } from '@bisonai/orakl-contracts'
import { State } from './state'
import { sendTransaction } from './utils'
import { watchman } from './watchman'
import {
  REPORTER_AGGREGATOR_QUEUE_NAME,
  BULLMQ_CONNECTION,
  CHAIN,
  DATA_FEED_REPORTER_STATE_NAME,
  DATA_FEED_SERVICE_NAME,
  PROVIDER_URL,
  DEPLOYMENT_NAME
} from '../settings'
import { IAggregatorWorkerReporter, IAggregatorHeartbeatWorker } from '../types'
import { OraklError, OraklErrorCode } from '../errors'
import { buildHeartbeatJobId } from '../utils'

const FILE_NAME = import.meta.url

export async function reporter(redisClient: RedisClientType, _logger: Logger) {
  const logger = _logger.child({ name: 'reporter', file: FILE_NAME })

  const state = new State({
    redisClient,
    providerUrl: PROVIDER_URL,
    stateName: DATA_FEED_REPORTER_STATE_NAME,
    service: DATA_FEED_SERVICE_NAME,
    chain: CHAIN,
    logger
  })
  await state.refresh()

  logger.debug(await state.active(), 'Active reporters')

  const DATA_FEED_REPORTER_WORKER_CONCURRENCY = 5
  const reporterWorker = new Worker(REPORTER_AGGREGATOR_QUEUE_NAME, await job(state, logger), {
    ...BULLMQ_CONNECTION,
    concurrency: DATA_FEED_REPORTER_WORKER_CONCURRENCY
  })
  reporterWorker.on('error', (e) => {
    logger.error(e)
  })

  const watchmanServer = await watchman({ state, logger })

  // Graceful shutdown
  async function handleExit() {
    logger.info('Exiting. Wait for graceful shutdown.')

    await redisClient.quit()
    await reporterWorker.close()
    await watchmanServer.close()
  }
  process.on('SIGINT', handleExit)
  process.on('SIGTERM', handleExit)

  logger.debug('Reporter launched')
}

function job(state: State, logger: Logger) {
  const iface = new ethers.utils.Interface(Aggregator__factory.abi)

  async function wrapper(job: Job) {
    const inData: IAggregatorWorkerReporter = job.data
    logger.debug(inData, 'inData')
    const { roundId, submission, oracleAddress } = inData

    const wallet = state.wallets[oracleAddress]
    if (!wallet) {
      const msg = `Wallet for oracle ${oracleAddress} is not active`
      logger.error(msg)
      throw new OraklError(OraklErrorCode.WalletNotActive, msg)
    }

    const payload = iface.encodeFunctionData('submit', [roundId, submission])
    const gasLimit = 300_000 // FIXME move to settings outside of code

    const NUM_TRANSACTION_TRIALS = 3
    for (let i = 0; i < NUM_TRANSACTION_TRIALS; ++i) {
      try {
        await sendTransaction({ wallet, to: oracleAddress, payload, logger, gasLimit })
        break
      } catch (e) {
        if (
          ![
            OraklErrorCode.TxNotMined,
            OraklErrorCode.TxProcessingResponseError,
            OraklErrorCode.TxMissingResponseError
          ].includes(e.code)
        ) {
          throw e
        }

        logger.info(`Retrying transaction. Trial number: ${i}`)
      }
    }
  }

  logger.debug('Reporter job built')
  return wrapper
}
