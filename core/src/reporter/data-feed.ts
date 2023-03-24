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
  HEARTBEAT_QUEUE_NAME,
  CHAIN,
  DATA_FEED_REPORTER_STATE_NAME,
  DATA_FEED_SERVICE_NAME,
  PROVIDER_URL
} from '../settings'
import { IAggregatorWorkerReporter, IAggregatorHeartbeatWorker } from '../types'
import { OraklError, OraklErrorCode } from '../errors'

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

  new Worker(REPORTER_AGGREGATOR_QUEUE_NAME, await job(state, logger), BULLMQ_CONNECTION)
  await watchman({ state, logger })
  logger.debug('Reporter worker launched')
}

function job(state: State, logger: Logger) {
  const iface = new ethers.utils.Interface(Aggregator__factory.abi)
  const heartbeatQueue = new Queue(HEARTBEAT_QUEUE_NAME, BULLMQ_CONNECTION)

  async function wrapper(job: Job) {
    const inData: IAggregatorWorkerReporter = job.data
    logger.debug(inData, 'inData')

    const oracleAddress = inData.callbackAddress
    await submitHeartbeatJob(heartbeatQueue, oracleAddress, inData.delay, logger)

    const wallet = state.wallets[oracleAddress]
    if (!wallet) {
      const msg = `Wallet for aggregator ${oracleAddress} is not active`
      logger.error(msg)
      throw new OraklError(OraklErrorCode.WalletNotActive, msg)
    }

    try {
      const payload = iface.encodeFunctionData('submit', [inData.roundId, inData.submission])
      const gasLimit = 300_000 // FIXME move to settings outside of code

      // TODO retry when transaction failed
      await sendTransaction({ wallet, to: oracleAddress, payload, logger, gasLimit })
    } catch (e) {
      logger.error(e)
      throw e
    }
  }

  logger.debug('Reporter job built')
  return wrapper
}

async function submitHeartbeatJob(
  heartbeatQueue: Queue,
  oracleAddress: string,
  delay: number,
  logger: Logger
) {
  const allDelayed = (await heartbeatQueue.getJobs(['delayed'])).filter(
    (job) => job.opts.jobId == oracleAddress
  )

  if (allDelayed.length > 1) {
    throw new OraklError(OraklErrorCode.UnexpectedNumberOfJobsInQueue)
  } else if (allDelayed.length == 1) {
    const delayedJob = allDelayed[0]
    delayedJob.remove()

    logger.debug({ job: 'deleted' }, 'job-deleted')
  }

  const outData: IAggregatorHeartbeatWorker = {
    aggregatorAddress: oracleAddress
  }
  await heartbeatQueue.add('heartbeat', outData, {
    delay: delay,
    removeOnComplete: true,
    jobId: oracleAddress,
    attempts: 3,
    backoff: 1_000
  })
  logger.debug({ job: 'added', delay: delay }, 'job-added')
  logger.debug('Reporter submitted heartbeat job')
}
