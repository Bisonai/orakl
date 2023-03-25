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
  PROVIDER_URL,
  HEARTBEAT_JOB_NAME,
  DEPLOYMENT_NAME,
  HEARTBEAT_QUEUE_SETTINGS
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

/**
 * Reported job might have been requested by [event] worker or
 * [deviation] worker before the end of heartbeat delay. If that is
 * the case, there is still waiting delayed heartbeat job in the
 * heartbeat queue. If that is the case, we remove it. Then, we submit
 * the new heartbeat job.
 *
 * @param {Queue} heartbeat queue
 * @param {string} oracle address
 * @param {delay} heartbeat delay
 * @param {Logger} pino logger
 * @return {void}
 */
async function submitHeartbeatJob(
  heartbeatQueue: Queue,
  oracleAddress: string,
  delay: number,
  logger: Logger
) {
  const jobId = buildHeartbeatJobId({ oracleAddress, deploymentName: DEPLOYMENT_NAME })
  const allDelayed = (await heartbeatQueue.getJobs(['delayed'])).filter(
    (job) => job.opts.jobId == jobId
  )

  if (allDelayed.length > 1) {
    throw new OraklError(OraklErrorCode.UnexpectedNumberOfJobsInQueue)
  } else if (allDelayed.length == 1) {
    const delayedJob = allDelayed[0]
    delayedJob.remove()

    logger.debug({ job: 'deleted' }, `Reporter deleted heartbeat job with ID=${jobId}`)
  }

  const jobData: IAggregatorHeartbeatWorker = {
    oracleAddress
  }
  await heartbeatQueue.add(HEARTBEAT_JOB_NAME, jobData, {
    jobId,
    delay,
    ...HEARTBEAT_QUEUE_SETTINGS
  })

  logger.debug({ job: 'added', delay }, `Reporter submitted heartbeat job with ID=${jobId}`)
}
