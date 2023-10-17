import { ethers } from 'ethers'
import { Worker, Queue, Job } from 'bullmq'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { Aggregator__factory } from '@bisonai/orakl-contracts'
import {
  getAggregatorGivenAddress,
  getAggregators,
  fetchDataFeed,
  getL2AddressGivenL1Address
} from './api'
import { State } from './state'
import { OraklError, OraklErrorCode } from '../errors'
import {
  IDataFeedListenerWorker,
  IAggregatorHeartbeatWorker,
  IAggregatorSubmitHeartbeatWorker,
  QueueType
} from '../types'
import {
  BULLMQ_CONNECTION,
  CHAIN,
  DATA_FEED_WORKER_STATE_NAME,
  DEPLOYMENT_NAME,
  HEARTBEAT_QUEUE_NAME,
  REMOVE_ON_COMPLETE,
  REPORTER_AGGREGATOR_QUEUE_NAME,
  SUBMIT_HEARTBEAT_QUEUE_NAME,
  SUBMIT_HEARTBEAT_QUEUE_SETTINGS,
  WORKER_AGGREGATOR_QUEUE_NAME,
  WORKER_CHECK_HEARTBEAT_QUEUE_NAME,
  MAX_DATA_STALENESS,
  DATA_FEED_FULFILL_GAS_MINIMUM
} from '../settings'
import { buildSubmissionRoundJobId, buildHeartbeatJobId } from '../utils'
import {
  oracleRoundStateCall,
  buildTransaction,
  isStale,
  getRoundDataCall
} from './data-feed.utils'
import { watchman } from './watchman'
import { getOperatorAddress } from '../api'
import { IDeviationData } from './types'

const FILE_NAME = import.meta.url

/**
 * Get all active aggregators, create their initial jobs, and submit
 * them to the [heartbeat] queue. Launch [aggregator] and [heartbeat]
 * workers.
 *
 * @param {RedisClientType} redis client
 * @param {Logger} pino logger
 */
export async function worker(redisClient: RedisClientType, _logger: Logger) {
  const logger = _logger.child({ name: 'worker', file: FILE_NAME })

  // Queues
  const heartbeatQueue = new Queue(HEARTBEAT_QUEUE_NAME, BULLMQ_CONNECTION)
  const submitHeartbeatQueue = new Queue(SUBMIT_HEARTBEAT_QUEUE_NAME, BULLMQ_CONNECTION)
  const reporterQueue = new Queue(REPORTER_AGGREGATOR_QUEUE_NAME, BULLMQ_CONNECTION)
  const state = new State({
    redisClient,
    stateName: DATA_FEED_WORKER_STATE_NAME,
    heartbeatQueue,
    submitHeartbeatQueue,
    chain: CHAIN,
    logger
  })
  await state.clear()

  const activeAggregators = await getAggregators({ chain: CHAIN, active: true, logger })
  if (activeAggregators.length == 0) {
    logger.warn('No active aggregators')
  }

  // Launch all active aggregators
  for (const aggregator of activeAggregators) {
    await state.add(aggregator.aggregatorHash)
  }

  // [aggregator] worker
  const aggregatorWorker = new Worker(
    WORKER_AGGREGATOR_QUEUE_NAME,
    aggregatorJob(reporterQueue, _logger),
    {
      ...BULLMQ_CONNECTION
    }
  )
  aggregatorWorker.on('error', (e) => {
    logger.error(e)
  })

  const watchmanServer = await watchman({ state, logger })

  async function handleExit() {
    logger.info('Exiting. Wait for graceful shutdown.')

    await redisClient.quit()
    await aggregatorWorker.close()
    await watchmanServer.close()
  }
  process.on('SIGINT', handleExit)
  process.on('SIGTERM', handleExit)
}

/**
 * [aggregator] worker receives both [event] and [heartbeat]
 * jobs. {event} jobs are created by listener. {heartbeat} jobs are
 * either created during a launch of a worker, or inside of a reporter.
 *
 * Worker accepts job, parses the request, fetches the latest
 * aggregated data from the Orakl Network API for a specific
 * aggregator, and communicated with Aggregator smart contract to find
 * out the which round ID, it can submit the latest value. Then, it
 * create a new job and passes it to reporter worker.
 *
 * @param {QueueType} submit heartbeat queue
 * @param {QueueType} reporter queue
 * @param {Logger} pino logger
 * @return {} [aggregator] job processor
 */
export function aggregatorJob(reporterQueue: QueueType, _logger: Logger) {
  const logger = _logger.child({ name: 'aggregatorJob' })
  const iface = new ethers.utils.Interface(Aggregator__factory.abi)

  async function wrapper(job: Job) {
    const inData: IDataFeedListenerWorker = job.data
    logger.debug(inData, 'inData')
    const { oracleAddress, roundId, workerSource } = inData

    try {
      // TODO store in ephemeral state
      const { aggregatorHash, heartbeat: delay } = await getAggregatorGivenAddress({
        oracleAddress,
        logger
      })

      const { l2OracleAddress } = await getL2AddressGivenL1Address({
        oracleAddress,
        logger
      })

      const { answer } = await getRoundDataCall({ oracleAddress, roundId })
      logger.debug({ aggregatorHash, fetchedData: answer }, 'Latest data')

      const tx = buildTransaction({
        payloadParameters: {
          roundId,
          submission: answer.toBigInt()
        },
        to: l2OracleAddress,
        gasMinimum: DATA_FEED_FULFILL_GAS_MINIMUM,
        iface,
        logger
      })
      logger.debug(tx, 'tx')

      await reporterQueue.add(workerSource, tx, {
        jobId: buildSubmissionRoundJobId({
          oracleAddress,
          roundId,
          deploymentName: DEPLOYMENT_NAME
        }),
        removeOnComplete: REMOVE_ON_COMPLETE,
        // Reporter job can fail, and should be either retried or
        // removed. We need to remove the job after repeated failure
        // to prevent deadlock when running with a single node operator.
        // After removing the job on failure, we can resubmit the job
        // with the same unique ID representing the submission for
        // specific aggregator on specific round.
        //
        // FIXME Rethink!
        removeOnFail: true
      })

      return tx
    } catch (e) {
      // `FailedToFetchFromDataFeed` exception can be raised from `prepareDataForReporter`.
      // `aggregatorJob` is being triggered by either `fixed` or `event` worker.
      // `event` job will not be resubmitted. `fixed` job might be
      // resubmitted, however due to the nature of fixed job cycle, the
      // resubmission might be delayed more than is acceptable. For this
      // reason jobs processed with `aggregatorJob` job must be retried with
      // appropriate logic.
      logger.error(e)
      throw e
    }
  }

  return wrapper
}
