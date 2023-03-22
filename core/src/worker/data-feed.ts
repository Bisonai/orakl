import { Worker, Queue, Job } from 'bullmq'
import { Logger } from 'pino'
import { getAggregatorGivenAddress, getActiveAggregators, fetchDataFeed } from './api'
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
 * them to the {heartbeat} queue. Launch {event} and {heartbeat}
 * workers.
 */
export async function worker(_logger: Logger) {
  const logger = _logger.child({ name: 'worker', file: FILE_NAME })
  const aggregators = await getActiveAggregators({ chain: CHAIN, logger })

  if (aggregators.length == 0) {
    logger.warn('No active aggregators')
    return 1
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

  // Event based worker
  new Worker(WORKER_AGGREGATOR_QUEUE_NAME, aggregatorJob(REPORTER_AGGREGATOR_QUEUE_NAME, _logger), {
    ...BULLMQ_CONNECTION,
    settings: {
      backoffStrategy: aggregatorJobBackOffStrategy
    }
  })

  // Fixed heartbeat worker
  new Worker(
    FIXED_HEARTBEAT_QUEUE_NAME,
    heartbeatJob(WORKER_AGGREGATOR_QUEUE_NAME, _logger),
    BULLMQ_CONNECTION
  )
}

/**
 * Aggregator worker receives both {event} and {heartbeat}
 * jobs. {event} jobs are created by listener. {heartbeat} jobs are
 * either created during a launch of a worker, or inside of a reporter.
 *
 * Worker accepts job, parses the request, fetches the latest
 * aggregated data from the Orakl Network API for a specific
 * aggregator, and communicated with Aggregator smart contract to find
 * out the which round ID, it can submit the latest value. Then, it
 * create a new job and passes it to reporter worker.
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
        await queue.add('fixed', outData, {
          removeOnComplete: REMOVE_ON_COMPLETE,
          removeOnFail: REMOVE_ON_FAIL,
          jobId: buildReporterJobId({ aggregatorAddress, roundId, deploymentName: DEPLOYMENT_NAME })
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
