import { Worker, Queue, Job } from 'bullmq'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { getAggregatorGivenAddress, getAggregators, fetchDataFeed } from './api'
import { State } from './state'
import { OraklError, OraklErrorCode } from '../errors'
import { IAggregatorWorker, IAggregatorWorkerReporter, IAggregatorHeartbeatWorker } from '../types'
import {
  WORKER_AGGREGATOR_QUEUE_NAME,
  REPORTER_AGGREGATOR_QUEUE_NAME,
  HEARTBEAT_QUEUE_NAME,
  BULLMQ_CONNECTION,
  DEPLOYMENT_NAME,
  REMOVE_ON_COMPLETE,
  CHAIN,
  HEARTBEAT_JOB_NAME,
  HEARTBEAT_QUEUE_SETTINGS,
  AGGREGATOR_QUEUE_SETTINGS,
  DATA_FEED_WORKER_STATE_NAME
} from '../settings'
import { buildSubmissionRoundJobId, buildHeartbeatJobId } from '../utils'
import { oracleRoundStateCall } from './data-feed.utils'
import { watchman } from './watchman'
import { getOperatorAddress } from './data-feed.utils'

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
  const heartbeatQueue = new Queue(HEARTBEAT_QUEUE_NAME, BULLMQ_CONNECTION)

  const state = new State({
    redisClient,
    stateName: DATA_FEED_WORKER_STATE_NAME,
    heartbeatQueue,
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
    aggregatorJob(REPORTER_AGGREGATOR_QUEUE_NAME, _logger),
    {
      ...BULLMQ_CONNECTION,
      settings: {
        backoffStrategy: aggregatorJobBackOffStrategy
      }
    }
  )

  aggregatorWorker.on('error', (e) => {
    logger.error(e)
  })

  // [heartbeat] worker
  const heartbeatWorker = new Worker(
    HEARTBEAT_QUEUE_NAME,
    heartbeatJob(WORKER_AGGREGATOR_QUEUE_NAME, state, _logger),
    BULLMQ_CONNECTION
  )

  heartbeatWorker.on('error', (e) => {
    logger.error(e)
  })

  heartbeatWorker.on('failed', (job: Job, error: Error) => {
    // Do something with the return value.
  })

  await watchman({ state, logger })
  logger.debug('Worker launched')
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
 * @param {string} reporter queue name
 * @param {Logger} pino logger
 * @return {} [aggregator] job processor
 */
function aggregatorJob(reporterQueueName: string, _logger: Logger) {
  const logger = _logger.child({ name: 'aggregatorJob', file: FILE_NAME })
  const reporterQueue = new Queue(reporterQueueName, BULLMQ_CONNECTION)

  async function wrapper(job: Job) {
    const inData: IAggregatorWorker = job.data
    logger.debug(inData, 'inData-regular')
    const oracleAddress = inData.oracleAddress
    const roundId = inData.roundId

    try {
      const operatorAddress = await getOperatorAddress({ oracleAddress, logger })
      const { aggregatorHash, heartbeat } = await getAggregatorGivenAddress({
        oracleAddress,
        logger
      })

      const outData = await prepareDataForReporter({
        aggregatorHash,
        oracleAddress,
        operatorAddress,
        report: true,
        workerSource: inData.workerSource,
        delay: heartbeat,
        roundId,
        logger
      })

      logger.debug(outData, 'outData-regular')

      await reporterQueue.add(inData.workerSource, outData, {
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
        removeOnFail: true
      })
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

/**
 * [heartbeat] worker receives job either from the launch of the Data
 * Feed Worker service, or from the Data Feed Reporter service. In
 * both cases heartbeat job is delayed by the `heartbeat` amount of
 * time specified in milliseconds.
 *
 * [heartbeat] job execution is independent of [event] job, however
 * only one of them is eligible to submit to Aggregator smart contract
 * for a specific round ID.
 *
 * At first, [heartbeat] worker finds out what round is currently
 * accepting submissions given an operator address extracted from
 * associated aggregator address. Then, it creates a new job with a
 * unique ID denoting the request for report on a specific
 * round. Finally, it submits the job to the [aggregator] worker.  The
 * job ID is created with the same format as in Data Feed Listener
 * service, which protects the [aggregator] worker from processing the
 * same request twice.
 *
 * @params {string} name of queue processed by [aggregator] worker
 * @params {Logger} pino logger
 * @return {} [heartbeat] job processor
 */
function heartbeatJob(aggregatorJobQueueName: string, state: State, _logger: Logger) {
  const logger = _logger.child({ name: 'heartbeatJob', file: FILE_NAME })
  const aggregatorQueue = new Queue(aggregatorJobQueueName, BULLMQ_CONNECTION)
  const reporterQueue = new Queue(REPORTER_AGGREGATOR_QUEUE_NAME, BULLMQ_CONNECTION)

  async function wrapper(job: Job) {
    try {
      const { oracleAddress } = job.data
      logger.debug(oracleAddress, 'oracleAddress-fixed')

      // [hearbeat] worker can be controlled by watchman which can
      // either activate or deactive a [heartbeat] job. When
      // [heartbeat] job cannot be found in a local aggregator state,
      // the job is assumed to be terminated, and worker will drop any
      // incoming job that should be performed on aggregator denoted
      // by `aggregatorAddress`.
      if (!state.isActive({ oracleAddress })) {
        logger.warn(`Heartbeat job for oracle ${oracleAddress} is no longer active. Exiting.`)
        return 0
      }

      const operatorAddress = await getOperatorAddress({ oracleAddress, logger })
      const oracleRoundState = await oracleRoundStateCall({
        oracleAddress,
        operatorAddress,
        logger
      })
      logger.debug(oracleRoundState, 'oracleRoundState-fixed')

      const roundId = oracleRoundState._roundId

      const outData: IAggregatorWorker = {
        oracleAddress,
        roundId: roundId,
        workerSource: 'fixed'
      }
      logger.debug(outData, 'outData-fixed')

      if (oracleRoundState._eligibleToSubmit) {
        logger.debug({ job: 'added', eligible: true, roundId }, 'before-eligible-fixed')

        const jobId = buildSubmissionRoundJobId({
          oracleAddress,
          roundId,
          deploymentName: DEPLOYMENT_NAME
        })

        // [heartbeat] worker is executed at predefined intervals and
        // is of vital importance for repeated submission to
        // Aggregator smart contract. [heartbeat] worker is not executed
        // earlier than N miliseconds (also called as a heartbeat) after
        // the latest submission. If the Aggregator smart contract
        // tells us that we are eligible to submit to `roundId`, it
        // means that reporter has not submitted any value there yet.
        // It also means there was no submission in the last N milliseconds.
        // If we happen to be at that situation, we assume there is
        // a deadlock and the Orakl Network Reporter service failed on
        // to submit on particular `roundId`.
        await removeAggregatorDeadlock(aggregatorQueue, jobId, logger)

        await aggregatorQueue.add('fixed', outData, {
          jobId,
          removeOnComplete: REMOVE_ON_COMPLETE,
          ...AGGREGATOR_QUEUE_SETTINGS
        })
        logger.debug({ job: 'added', eligible: true, roundId }, 'eligible-fixed')
      } else {
        const msg = `Non-eligible to submit for oracle ${oracleAddress} with operator ${operatorAddress}`
        throw new OraklError(OraklErrorCode.NonEligibleToSubmit, msg)
      }
    } catch (e) {
      logger.error(e)
      throw e
    }
  }

  return wrapper
}

/**
 * Fetch the latest data and prepare them to be sent to reporter.
 *
 * @param {string} id: aggregator ID
 * @param {string} oracle address
 * @param {boolean} report: whether to submission must be reported
 * @param {string} workerSource
 * @param {number} delay
 * @param {number} roundId
 * @param {Logger} _logger
 * @return {Promise<IAggregatorJob}
 * @exception {FailedToFetchFromDataFeed} raised from `fetchDataFeed`
 */
async function prepareDataForReporter({
  aggregatorHash,
  oracleAddress,
  operatorAddress,
  report,
  workerSource,
  delay,
  roundId,
  logger
}: {
  aggregatorHash: string
  oracleAddress: string
  operatorAddress: string
  report?: boolean
  workerSource: string
  delay: number
  roundId?: number
  logger: Logger
}): Promise<IAggregatorWorkerReporter> {
  logger.debug('prepareDataForReporter')

  const { value } = await fetchDataFeed({ aggregatorHash, logger })

  const oracleRoundState = await oracleRoundStateCall({
    oracleAddress,
    operatorAddress,
    roundId,
    logger
  })
  logger.debug(oracleRoundState, 'oracleRoundState')

  return {
    report,
    callbackAddress: oracleAddress,
    workerSource,
    delay,
    submission: value,
    roundId: roundId || oracleRoundState._roundId
  }
}

function aggregatorJobBackOffStrategy(
  attemptsMade: number,
  type: string,
  err: Error,
  job: Job
): number {
  // TODO stop if there is newer job submitted
  return 1_000
}

/**
 * Remove aggregator deadlock: The job has already been requested and
 * accepted from the other end of queue, however, the job might not
 * have been accomplished successfully there. The function deletes the
 * previously submitted job, so it can be resubmitted again.
 *
 * Note: This function should be called only when we are certain that
 * there is any deadlock. Deadlock detection is not part of this
 * function.
 *
 * @param {queue} aggregator queue
 * @param {string} job ID
 * @param {Logger} pino logger
 * @return {void}
 * @except {OraklErrorCode.UnexpectedNumberOfDeadlockJobs} raise when
 * more than single deadlock found
 */
async function removeAggregatorDeadlock(aggregatorQueue: Queue, jobId: string, logger: Logger) {
  const blockingJob = (await aggregatorQueue.getJobs(['completed'])).filter(
    (job) => job.opts.jobId == jobId
  )

  if (blockingJob.length == 1) {
    blockingJob[0].remove()
    logger.warn(`Removed blocking job with ID ${jobId}`)
  } else if (blockingJob.length > 1) {
    throw new OraklError(
      OraklErrorCode.UnexpectedNumberOfDeadlockJobs,
      `Found ${blockingJob.length} blocking jobs. Expected 1 at most.`
    )
  }
}
