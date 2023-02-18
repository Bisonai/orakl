import { Worker, Queue } from 'bullmq'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import {
  IAggregatorListenerWorker,
  IAggregatorWorkerReporter,
  IAggregatorHeartbeatWorker
} from '../types'
import {
  WORKER_AGGREGATOR_QUEUE_NAME,
  REPORTER_AGGREGATOR_QUEUE_NAME,
  FIXED_HEARTBEAT_QUEUE_NAME,
  RANDOM_HEARTBEAT_QUEUE_NAME,
  BULLMQ_CONNECTION,
  PUBLIC_KEY as OPERATOR_ADDRESS,
  REDIS_HOST,
  REDIS_PORT,
  lastSubmissionTimeKey,
  toSubmitTimeKey
} from '../settings'
import { IcnError, IcnErrorCode } from '../errors'
import { createRedisClient } from '../utils'
import {
  fetchDataWithAdapter,
  loadAdapters,
  loadAggregators,
  mergeAggregatorsAdapters,
  uniform,
  oracleRoundStateCall
} from './utils'

const FILE_NAME = import.meta.url

// FIXME move to settings?
const REMOVE_ON_COMPLETE = 500
const REMOVE_ON_FAIL = 1_000

export async function aggregatorWorker(_logger: Logger) {
  const logger = _logger.child({ name: 'aggregatorWorker', file: FILE_NAME })

  const adapters = await loadAdapters({ postprocess: true })
  logger.debug(adapters, 'adapters')

  const aggregators = await loadAggregators({ postprocess: true })
  logger.debug(aggregators, 'aggregators')

  const aggregatorsWithAdapters = mergeAggregatorsAdapters(aggregators, adapters)
  logger.debug(aggregatorsWithAdapters, 'aggregatorsWithAdapters')

  const redisClient = await createRedisClient(REDIS_HOST, REDIS_PORT)

  // Event based worker
  new Worker(
    WORKER_AGGREGATOR_QUEUE_NAME,
    aggregatorJob(REPORTER_AGGREGATOR_QUEUE_NAME, aggregatorsWithAdapters, _logger),
    BULLMQ_CONNECTION
  )

  // Fixed heartbeat worker
  new Worker(
    FIXED_HEARTBEAT_QUEUE_NAME,
    fixedHeartbeatJob(
      FIXED_HEARTBEAT_QUEUE_NAME,
      REPORTER_AGGREGATOR_QUEUE_NAME,
      aggregatorsWithAdapters,
      redisClient,
      _logger
    ),
    BULLMQ_CONNECTION
  )

  // Random heartbeat worker
  new Worker(
    RANDOM_HEARTBEAT_QUEUE_NAME,
    randomHeartbeatJob(
      RANDOM_HEARTBEAT_QUEUE_NAME,
      REPORTER_AGGREGATOR_QUEUE_NAME,
      aggregatorsWithAdapters,
      _logger
    ),
    BULLMQ_CONNECTION
  )
}

function aggregatorJob(reporterQueueName: string, aggregatorsWithAdapters, _logger: Logger) {
  const logger = _logger.child({ name: 'aggregatorJob', file: FILE_NAME })
  // This job is coming from on-chain request (event NewRound). Oracle
  // needs to submit the latest data based on this request without any
  // check on time or data change.
  const reporterQueue = new Queue(reporterQueueName, BULLMQ_CONNECTION)

  async function wrapper(job) {
    const inData: IAggregatorListenerWorker = job.data
    logger.debug(inData, 'inData')

    if (!aggregatorsWithAdapters[inData.address]) {
      const msg = `Address not found in aggregators ${inData.address}`
      logger.error(msg)
      throw new IcnError(IcnErrorCode.AggregatorNotFound, msg)
    }

    // TODO check on existence
    const ag = addReportProperty(aggregatorsWithAdapters[inData.address], true)

    try {
      const outData = await prepareDataForReporter({ data: ag, workerSource: 'regular', _logger })
      logger.debug(outData, 'outData')
      reporterQueue.add('aggregator', outData, {
        removeOnComplete: true
      })
    } catch (e) {
      logger.error(e)
    }
  }

  return wrapper
}

function fixedHeartbeatJob(
  heartbeatQueueName: string,
  reporterQueueName: string,
  agregatorsWithAdapters: IAggregatorHeartbeatWorker[],
  redisClient: RedisClientType,
  _logger: Logger
) {
  const logger = _logger.child({ name: 'fixedHeartBeatJob', file: FILE_NAME })

  const heartbeatQueue = new Queue(heartbeatQueueName, BULLMQ_CONNECTION)
  const reporterQueue = new Queue(reporterQueueName, BULLMQ_CONNECTION)

  // Launch all aggregators to be executed with fixed heartbeat
  for (const k in agregatorsWithAdapters) {
    const ag = agregatorsWithAdapters[k]
    if (ag.fixedHeartbeatRate.active) {
      heartbeatQueue.add('fixed-heartbeat', addReportProperty(ag, true), {
        delay: ag.fixedHeartbeatRate.value,
        removeOnComplete: REMOVE_ON_COMPLETE,
        removeOnFail: REMOVE_ON_FAIL
      })
    }
  }

  async function wrapper(job) {
    const inData: IAggregatorHeartbeatWorker = job.data
    logger.debug(inData, 'inData')

    const aggregatorAddress = inData.address
    const now = Date.now()
    const lastSubmissionTime = Number(
      await redisClient.get(lastSubmissionTimeKey(aggregatorAddress))
    )
    const toSubmitTime = Number(await redisClient.get(toSubmitTimeKey(aggregatorAddress)))
    const isFirstSubmission = lastSubmissionTime == 0

    let slept = false
    try {
      if (
        isFirstSubmission ||
        Math.max(lastSubmissionTime, toSubmitTime) + inData.fixedHeartbeatRate.value <= now
      ) {
        await redisClient.set(toSubmitTimeKey(aggregatorAddress), now)

        const outData = await prepareDataForReporter({
          data: inData,
          workerSource: 'fixed',
          _logger
        })
        logger.debug(outData, 'outData')

        if (outData.report) {
          reporterQueue.add('aggregator', outData, {
            removeOnComplete: REMOVE_ON_COMPLETE,
            removeOnFail: REMOVE_ON_FAIL,
            jobId: buildReporterJobId({ aggregatorAddress, ...outData })
          })
        }
      } else {
        logger.debug('Sleep more')
        slept = true
      }
    } catch (e) {
      logger.error(e)
    } finally {
      const delay = slept
        ? Math.max(
            0,
            inData.fixedHeartbeatRate.value - (now - Math.max(lastSubmissionTime, toSubmitTime))
          )
        : inData.fixedHeartbeatRate.value

      logger.debug({ delay }, 'delay') // TODO delete

      heartbeatQueue.add('fixed-heartbeat', inData, {
        delay,
        removeOnComplete: REMOVE_ON_COMPLETE,
        removeOnFail: REMOVE_ON_FAIL
      })
    }
  }

  return wrapper
}

function randomHeartbeatJob(
  heartbeatQueueName: string,
  reporterQueueName: string,
  agregatorsWithAdapters: IAggregatorHeartbeatWorker[],
  _logger: Logger
) {
  const logger = _logger.child({ name: 'randomHeartbeatJob', file: FILE_NAME })

  const heartbeatQueue = new Queue(heartbeatQueueName, BULLMQ_CONNECTION)
  const reporterQueue = new Queue(reporterQueueName, BULLMQ_CONNECTION)

  // Launch all aggregators to be executed with random heartbeat
  for (const k in agregatorsWithAdapters) {
    const ag = agregatorsWithAdapters[k]
    if (ag.randomHeartbeatRate.active) {
      heartbeatQueue.add('random-heartbeat', addReportProperty(ag, undefined), {
        delay: uniform(0, ag.randomHeartbeatRate.value),
        removeOnComplete: REMOVE_ON_COMPLETE,
        removeOnFail: REMOVE_ON_FAIL
      })
    }
  }

  async function wrapper(job) {
    const inData: IAggregatorHeartbeatWorker = job.data
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
        reporterQueue.add('aggregator', outData, {
          removeOnComplete: REMOVE_ON_COMPLETE,
          removeOnFail: REMOVE_ON_FAIL,
          jobId: buildReporterJobId({ aggregatorAddress, ...outData })
        })
      }
    } catch (e) {
      logger.error(e)
    } finally {
      heartbeatQueue.add('random-heartbeat', inData, {
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
  _logger
}: {
  data: IAggregatorHeartbeatWorker
  workerSource: string
  _logger: Logger
}): Promise<IAggregatorWorkerReporter> {
  const logger = _logger.child({ name: 'prepareDataForReporter', file: FILE_NAME })

  const callbackAddress = data.address
  const submission = await fetchDataWithAdapter(data.adapter)
  let report = data.report

  const oracleRoundState = await oracleRoundStateCall({
    aggregatorAddress: data.address,
    operatorAddress: OPERATOR_ADDRESS,
    logger
  })
  logger.debug(oracleRoundState, 'oracleRoundState')

  if (report === undefined) {
    const latestSubmission = oracleRoundState._latestSubmission.toNumber()
    report = shouldReport(latestSubmission, submission, data.threshold, data.absoluteThreshold)
    logger.debug({ report }, 'report')
  }

  return {
    report,
    callbackAddress,
    workerSource,
    submission,
    roundId: oracleRoundState._roundId
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
  threshold: number,
  absoluteThreshold: number
): boolean {
  if (latestSubmission && submission) {
    const range = latestSubmission * threshold
    const l = latestSubmission - range
    const r = latestSubmission + range
    return submission < l || submission > r
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

function buildReporterJobId({
  aggregatorAddress,
  roundId
}: {
  aggregatorAddress: string
  roundId: number
}) {
  return `${roundId}-${aggregatorAddress}`
}
