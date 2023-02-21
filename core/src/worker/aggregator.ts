import { Worker, Queue } from 'bullmq'
import { Logger } from 'pino'
import {
  IAggregatorWorker,
  IAggregatorWorkerReporter,
  IAggregatorHeartbeatWorker,
  IAggregatorJob
} from '../types'
import {
  WORKER_AGGREGATOR_QUEUE_NAME,
  REPORTER_AGGREGATOR_QUEUE_NAME,
  FIXED_HEARTBEAT_QUEUE_NAME,
  RANDOM_HEARTBEAT_QUEUE_NAME,
  BULLMQ_CONNECTION,
  PUBLIC_KEY as OPERATOR_ADDRESS,
  DEPLOYMENT_NAME,
  REMOVE_ON_COMPLETE,
  REMOVE_ON_FAIL
} from '../settings'
import { IcnError, IcnErrorCode } from '../errors'
import { buildReporterJobId } from '../utils'
import {
  fetchDataWithAdapter,
  loadAdapters,
  loadAggregators,
  mergeAggregatorsAdapters,
  uniform,
  oracleRoundStateCall
} from './utils'

const FILE_NAME = import.meta.url

export async function aggregatorWorker(_logger: Logger) {
  const logger = _logger.child({ name: 'aggregatorWorker', file: FILE_NAME })

  const adapters = await loadAdapters({ postprocess: true })
  logger.debug(adapters, 'adapters')

  const aggregators = await loadAggregators({ postprocess: true })
  logger.debug(aggregators, 'aggregators')

  const aggregatorsWithAdapters = mergeAggregatorsAdapters(aggregators, adapters)
  logger.debug(aggregatorsWithAdapters, 'aggregatorsWithAdapters')

  // Launch all aggregators to be executed with random heartbeat
  const heartbeatQueue = new Queue(RANDOM_HEARTBEAT_QUEUE_NAME, BULLMQ_CONNECTION)
  for (const aggregatorAddress in aggregatorsWithAdapters) {
    const aggregator = aggregatorsWithAdapters[aggregatorAddress]
    if (aggregator.randomHeartbeatRate.active) {
      await heartbeatQueue.add('random-heartbeat', addReportProperty(aggregator, undefined), {
        delay: uniform(0, aggregator.randomHeartbeatRate.value),
        removeOnComplete: REMOVE_ON_COMPLETE,
        removeOnFail: REMOVE_ON_FAIL
      })
    }
  }

  // Event based worker
  new Worker(
    WORKER_AGGREGATOR_QUEUE_NAME,
    aggregatorJob(REPORTER_AGGREGATOR_QUEUE_NAME, aggregatorsWithAdapters, _logger),
    BULLMQ_CONNECTION
  )

  // Fixed heartbeat worker
  new Worker(
    FIXED_HEARTBEAT_QUEUE_NAME,
    fixedHeartbeatJob(WORKER_AGGREGATOR_QUEUE_NAME, _logger),
    BULLMQ_CONNECTION
  )

  // Random heartbeat worker
  new Worker(
    RANDOM_HEARTBEAT_QUEUE_NAME,
    randomHeartbeatJob(RANDOM_HEARTBEAT_QUEUE_NAME, REPORTER_AGGREGATOR_QUEUE_NAME, _logger),
    BULLMQ_CONNECTION
  )
}

function aggregatorJob(
  reporterQueueName: string,
  aggregatorsWithAdapters: IAggregatorJob[],
  _logger: Logger
) {
  const logger = _logger.child({ name: 'aggregatorJob', file: FILE_NAME })
  const reporterQueue = new Queue(reporterQueueName, BULLMQ_CONNECTION)

  async function wrapper(job) {
    const inData: IAggregatorWorker = job.data
    const aggregatorAddress = inData.aggregatorAddress
    const roundId = inData.roundId

    if (!aggregatorsWithAdapters[aggregatorAddress]) {
      throw new IcnError(IcnErrorCode.UndefinedAggregator)
    }

    try {
      const aggregator = addReportProperty(aggregatorsWithAdapters[aggregatorAddress], true)

      const outData = await prepareDataForReporter({
        data: aggregator,
        workerSource: inData.workerSource,
        roundId,
        _logger
      })
      logger.debug(outData, 'outData')

      await reporterQueue.add(inData.workerSource, outData, {
        removeOnComplete: REMOVE_ON_COMPLETE,
        removeOnFail: REMOVE_ON_FAIL,
        jobId: buildReporterJobId({
          aggregatorAddress,
          roundId,
          deploymentName: DEPLOYMENT_NAME
        })
      })
    } catch (e) {
      logger.error(e)
    }
  }

  return wrapper
}

function fixedHeartbeatJob(aggregatorJobQueueName: string, _logger: Logger) {
  const logger = _logger.child({ name: 'fixedHeartbeatJob', file: FILE_NAME })
  const queue = new Queue(aggregatorJobQueueName, BULLMQ_CONNECTION)

  async function wrapper(job) {
    const inData: IAggregatorHeartbeatWorker = job.data
    const aggregatorAddress = inData.aggregatorAddress

    try {
      const oracleRoundState = await oracleRoundStateCall({
        aggregatorAddress,
        operatorAddress: OPERATOR_ADDRESS,
        logger
      })

      const outData: IAggregatorWorker = {
        aggregatorAddress,
        roundId: oracleRoundState._roundId,
        workerSource: 'fixed'
      }

      if (oracleRoundState._eligibleToSubmit) {
        await queue.add('fixed', outData, {
          removeOnComplete: REMOVE_ON_COMPLETE,
          removeOnFail: REMOVE_ON_FAIL
        })
      }
    } catch (e) {
      logger.error(e)
    }
  }

  return wrapper
}

function randomHeartbeatJob(
  heartbeatQueueName: string,
  reporterQueueName: string,
  _logger: Logger
) {
  const logger = _logger.child({ name: 'randomHeartbeatJob', file: FILE_NAME })

  const heartbeatQueue = new Queue(heartbeatQueueName, BULLMQ_CONNECTION)
  const reporterQueue = new Queue(reporterQueueName, BULLMQ_CONNECTION)

  async function wrapper(job) {
    const inData: IAggregatorJob = job.data
    logger.debug(inData, 'inData')

    const aggregatorAddress = inData.address

    try {
      const outData = await prepareDataForReporter({
        data: inData,
        workerSource: 'random',
        _logger
      })
      logger.debug(outData, 'outData')
      if (outData.report) {
        await reporterQueue.add('random', outData, {
          removeOnComplete: REMOVE_ON_COMPLETE,
          removeOnFail: REMOVE_ON_FAIL,
          jobId: buildReporterJobId({
            aggregatorAddress,
            deploymentName: DEPLOYMENT_NAME,
            ...outData
          })
        })
      }
    } catch (e) {
      logger.error(e)
    } finally {
      await heartbeatQueue.add('random-heartbeat', inData, {
        delay: uniform(0, inData.randomHeartbeatRate.value),
        removeOnComplete: REMOVE_ON_COMPLETE,
        removeOnFail: REMOVE_ON_FAIL
      })
    }
  }

  return wrapper
}

/**
 * Fetch the latest data and prepare them to be sent to reporter.
 *
 * @param {IAggregatorHeartbeatWorker} data
 * @return {Promise<IAggregatorWorkerReporter>}
 * @exception {InvalidPriceFeed} raised from `fetchDataWithadapter`
 */
async function prepareDataForReporter({
  data,
  workerSource,
  roundId,
  _logger
}: {
  data: IAggregatorJob
  workerSource: string
  roundId?: number
  _logger: Logger
}): Promise<IAggregatorWorkerReporter> {
  const logger = _logger.child({ name: 'prepareDataForReporter', file: FILE_NAME })

  const callbackAddress = data.address
  const submission = await fetchDataWithAdapter(data.adapter)
  let report = data.report

  const oracleRoundState = await oracleRoundStateCall({
    aggregatorAddress: data.address,
    operatorAddress: OPERATOR_ADDRESS,
    roundId,
    logger
  })
  logger.debug(oracleRoundState, 'oracleRoundState')

  if (report === undefined) {
    // TODO does _latestsubmission hold the aggregated value?
    const latestSubmission = oracleRoundState._latestSubmission.toNumber()
    report = shouldReport(
      latestSubmission,
      submission,
      data.decimals,
      data.threshold,
      data.absoluteThreshold
    )
    logger.debug({ report }, 'report')
  }

  return {
    report,
    callbackAddress,
    workerSource,
    submission,
    roundId: roundId || oracleRoundState._roundId
  }
}

/**
 * Test whether the current submission deviates from the last
 * submission more than given threshold or absolute threshold. If yes,
 * return `true`, otherwise `false`.
 *
 * @param {number} latestSubmission
 * @param {number} submission
 * @param {number} threshold
 * @param {number} absolutethreshold
 * @return {boolean}
 */
function shouldReport(
  latestSubmission: number,
  submission: number,
  decimals: number,
  threshold: number,
  absoluteThreshold: number
): boolean {
  if (latestSubmission && submission) {
    const denominator = Math.pow(10, decimals)
    const latestSubmissionReal = latestSubmission / denominator
    const submissionReal = submission / denominator

    const range = latestSubmissionReal * threshold
    const l = latestSubmissionReal - range
    const r = latestSubmissionReal + range
    return submissionReal < l || submissionReal > r
  } else if (!latestSubmission && submission) {
    // latestSubmission hit zero
    return submission > absoluteThreshold
  } else {
    // Something strange happened, don't report!
    return false
  }
}

function addReportProperty(o, report: boolean | undefined) {
  return Object.assign({}, ...[o, { report }])
}
