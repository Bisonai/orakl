import { Worker, Queue, Job } from 'bullmq'
import { Logger } from 'pino'
import { getAggregatorGivenAddress, getActiveAggregators, fetchDataFeed } from './api'
import { OraklError, OraklErrorCode } from '../errors'
import { getReporterByOracleAddress } from '../api'
import { IAggregatorWorker, IAggregatorWorkerReporter } from '../types'
import {
  WORKER_AGGREGATOR_QUEUE_NAME,
  REPORTER_AGGREGATOR_QUEUE_NAME,
  FIXED_HEARTBEAT_QUEUE_NAME,
  BULLMQ_CONNECTION,
  DEPLOYMENT_NAME,
  REMOVE_ON_COMPLETE,
  REMOVE_ON_FAIL,
  CHAIN,
  DATA_FEED_SERVICE_NAME
} from '../settings'
import { buildReporterJobId } from '../utils'
import { oracleRoundStateCall } from './utils'

const FILE_NAME = import.meta.url

/**
 * Get all active aggregators, create their initial jobs, and submit
 * them to the [heartbeat] queue. Launch [event] and [heartbeat]
 * workers.
 *
 * @param {Logger} pino logger
 */
export async function worker(_logger: Logger) {
  const logger = _logger.child({ name: 'worker', file: FILE_NAME })
  const aggregators = await getActiveAggregators({ chain: CHAIN, logger })

  if (aggregators.length == 0) {
    logger.warn('No active aggregators')
  }

  // Launch all active aggregators
  const fixedHeartbeatQueue = new Queue(FIXED_HEARTBEAT_QUEUE_NAME, BULLMQ_CONNECTION)
  for (const aggregator of aggregators) {
    const aggregatorAddress = aggregator.address

    const operatorAddress = await getOperatorAddress({ oracleAddress: aggregatorAddress, logger })
    await fixedHeartbeatQueue.add(
      'heartbeat',
      { aggregatorAddress },
      {
        delay: await getSynchronizedDelay({
          aggregatorAddress,
          operatorAddress,
          heartbeat: aggregator.heartbeat,
          logger
        }),
        removeOnComplete: true,
        removeOnFail: true
      }
    )
  }

  // {event}  worker
  new Worker(WORKER_AGGREGATOR_QUEUE_NAME, aggregatorJob(REPORTER_AGGREGATOR_QUEUE_NAME, _logger), {
    ...BULLMQ_CONNECTION,
    settings: {
      backoffStrategy: aggregatorJobBackOffStrategy
    }
  })

  // {heartbeat} worker
  new Worker(
    FIXED_HEARTBEAT_QUEUE_NAME,
    heartbeatJob(WORKER_AGGREGATOR_QUEUE_NAME, _logger),
    BULLMQ_CONNECTION
  )
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
    const aggregatorAddress = inData.aggregatorAddress
    const roundId = inData.roundId

    try {
      const operatorAddress = await getOperatorAddress({ oracleAddress: aggregatorAddress, logger })
      const { aggregatorHash, heartbeat } = await getAggregatorGivenAddress({
        aggregatorAddress,
        logger
      })

      const outData = await prepareDataForReporter({
        aggregatorHash,
        aggregatorAddress,
        operatorAddress,
        report: true,
        workerSource: inData.workerSource,
        delay: heartbeat,
        roundId,
        logger
      })

      logger.debug(outData, 'outData-regular')

      await reporterQueue.add(inData.workerSource, outData, {
        removeOnComplete: REMOVE_ON_COMPLETE,
        // Reporter job can fail, and should be either retried or
        // removed. We need to remove the job after repeated failure
        // to prevent deadlock when running with a single node operator.
        // After removing the job on failure, we can resubmit the job
        // with the same unique ID representing the submission for
        // specific aggregator on specific round.
        removeOnFail: true,
        jobId: buildReporterJobId({
          aggregatorAddress,
          roundId,
          deploymentName: DEPLOYMENT_NAME
        })
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
function heartbeatJob(aggregatorJobQueueName: string, _logger: Logger) {
  const logger = _logger.child({ name: 'heartbeatJob', file: FILE_NAME })
  const queue = new Queue(aggregatorJobQueueName, BULLMQ_CONNECTION)

  async function wrapper(job: Job) {
    const { aggregatorAddress } = job.data
    logger.debug(aggregatorAddress, 'aggregatorAddress-fixed')

    try {
      const operatorAddress = await getOperatorAddress({ oracleAddress: aggregatorAddress, logger })
      const oracleRoundState = await oracleRoundStateCall({
        aggregatorAddress,
        operatorAddress,
        logger
      })
      logger.debug(oracleRoundState, 'oracleRoundState-fixed')

      const roundId = oracleRoundState._roundId

      const outData: IAggregatorWorker = {
        aggregatorAddress,
        roundId: roundId,
        workerSource: 'fixed'
      }
      logger.debug(outData, 'outData-fixed')

      if (oracleRoundState._eligibleToSubmit) {
        logger.debug({ job: 'added', eligible: true, roundId }, 'before-eligible-fixed')

        const jobId = buildReporterJobId({
          aggregatorAddress,
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
        await removeDeadlock(queue, jobId, logger)

        await queue.add('fixed', outData, {
          removeOnComplete: REMOVE_ON_COMPLETE,
          removeOnFail: REMOVE_ON_FAIL,
          jobId
        })
        logger.debug({ job: 'added', eligible: true, roundId }, 'eligible-fixed')
      } else {
        logger.debug({ eligible: false, roundId }, 'non-eligible-fixed')
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
  aggregatorAddress,
  operatorAddress,
  report,
  workerSource,
  delay,
  roundId,
  logger
}: {
  aggregatorHash: string
  aggregatorAddress: string
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
    aggregatorAddress,
    operatorAddress,
    roundId,
    logger
  })
  logger.debug(oracleRoundState, 'oracleRoundState')

  return {
    report,
    callbackAddress: aggregatorAddress,
    workerSource,
    delay,
    submission: value,
    roundId: roundId || oracleRoundState._roundId
  }
}

/**
 * Compute the number of seconds until the next round.
 *
 * FIXME modify aggregator to use single contract call
 *
 * @param {string} aggregator address
 * @param {number} heartbeat
 * @param {Logger}
 * @return {number} delay in seconds until the next round
 */
async function getSynchronizedDelay({
  aggregatorAddress,
  operatorAddress,
  heartbeat,
  logger
}: {
  aggregatorAddress: string
  operatorAddress: string
  heartbeat: number
  logger: Logger
}): Promise<number> {
  logger.debug('getSynchronizedDelay')

  let startedAt = 0
  const { _startedAt, _roundId } = await oracleRoundStateCall({
    aggregatorAddress,
    operatorAddress,
    logger
  })

  if (_startedAt.toNumber() != 0) {
    startedAt = _startedAt.toNumber()
  } else {
    const { _startedAt } = await oracleRoundStateCall({
      aggregatorAddress,
      operatorAddress,
      roundId: Math.max(0, _roundId - 1)
    })
    startedAt = _startedAt.toNumber()
  }

  logger.debug({ startedAt }, 'startedAt')
  const delay = heartbeat - (startedAt % heartbeat)
  logger.debug({ delay }, 'delay')

  return delay
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
 * Get address of node operator given an `oracleAddress`. The data are fetched from the Orakl Network API.
 *
 * @param {string} oracle address
 * @return {string} address of node operator
 * @exception {OraklErrorCode.GetReporterRequestFailed} raises when request failed
 */
async function getOperatorAddress({
  oracleAddress,
  logger
}: {
  oracleAddress: string
  logger: Logger
}) {
  logger.debug('getOperatorAddress')

  return await (
    await getReporterByOracleAddress({
      service: DATA_FEED_SERVICE_NAME,
      chain: CHAIN,
      oracleAddress,
      logger
    })
  ).address
}

/**
 * Remove deadlock: The job has already been requested and accepted
 * from the other end of queue, however, the job might not have been
 * accomplished successfully there. The function deletes the
 * previously submitted job, so it can be resubmitted again.
 *
 * Note: This function should be called only when we are certain that
 * there is any deadlock. Deadlock detection is not part of this
 * function.
 *
 * @param {queue} queue
 * @param {string} job ID
 * @param {Logger} pino logger
 * @return {void}
 * @except {OraklErrorCode.UnexpectedNumberOfDeadlockJobs} raise when
 * more than single deadlock found
 */
async function removeDeadlock(queue: Queue, jobId: string, logger: Logger) {
  const blockingJob = (await queue.getJobs(['completed'])).filter((job) => job.opts.jobId == jobId)

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
