import { ethers } from 'ethers'
import { Worker, Queue } from 'bullmq'
import { Logger } from 'pino'
import {
  fetchDataWithAdapter,
  loadAdapters,
  loadAggregators,
  mergeAggregatorsAdapters,
  uniform
} from './utils'
import {
  IAggregatorListenerWorker,
  IAggregatorWorkerReporter,
  IAggregatorHeartbeatWorker,
  IOracleRoundState
} from '../types'
import {
  WORKER_AGGREGATOR_QUEUE_NAME,
  REPORTER_AGGREGATOR_QUEUE_NAME,
  FIXED_HEARTBEAT_QUEUE_NAME,
  RANDOM_HEARTBEAT_QUEUE_NAME,
  BULLMQ_CONNECTION
} from '../settings'
import { PROVIDER_URL, PUBLIC_KEY as ORACLE_ADDRESS } from '../settings'
import { Aggregator__factory } from '@bisonai-cic/icn-contracts'

const FILE_NAME = import.meta.url

export async function aggregatorWorker(_logger: Logger) {
  const logger = _logger.child({ name: 'aggregatorWorker', file: FILE_NAME })

  const adapters = await loadAdapters({ postprocess: true })
  logger.debug(adapters, 'adapters')

  const aggregators = await loadAggregators({ postprocess: true })
  logger.debug(aggregators, 'aggregators')

  const aggregatorsWithAdapters = mergeAggregatorsAdapters(aggregators, adapters)
  logger.debug(aggregatorsWithAdapters, 'aggregatorsWithAdapters')

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
    const ag = addReportProperty(aggregatorsWithAdapters[inData.address], true)

    try {
      const outData = await prepareDataForReporter(ag, _logger)
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
  _logger: Logger
) {
  const logger = _logger.child({ name: 'fixedHeartBeatJob', file: FILE_NAME })

  const heartbeatQueue = new Queue(heartbeatQueueName, BULLMQ_CONNECTION)
  const reporterQueue = new Queue(reporterQueueName, BULLMQ_CONNECTION)

  // Launch all aggregators to be executed with fixed heartbeat
  // TODO Add clock synchronization through on-chain public data timestamp
  for (const k in agregatorsWithAdapters) {
    const ag = agregatorsWithAdapters[k]
    if (ag.fixedHeartbeatRate.active) {
      heartbeatQueue.add('fixed-heartbeat', addReportProperty(ag, true), {
        delay: ag.fixedHeartbeatRate.value,
        removeOnComplete: true
      })
    }
  }

  async function wrapper(job) {
    const inData: IAggregatorHeartbeatWorker = job.data
    logger.debug(inData, 'inData')

    try {
      const outData = await prepareDataForReporter(inData, _logger)
      logger.debug(outData, 'outData')
      if (outData.report) {
        reporterQueue.add('aggregator', outData, { removeOnComplete: true })
      }
    } catch (e) {
      logger.error(e)
    } finally {
      heartbeatQueue.add('fixed-heartbeat', inData, {
        delay: inData.fixedHeartbeatRate.value,
        removeOnComplete: true
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
      heartbeatQueue.add('random-heartbeat', ag, {
        delay: uniform(0, ag.randomHeartbeatRate.value),
        removeOnComplete: true
      })
    }
  }

  async function wrapper(job) {
    const inData: IAggregatorHeartbeatWorker = job.data
    logger.debug(inData, 'inData')

    try {
      const outData = await prepareDataForReporter(inData, _logger)
      logger.debug(outData, 'outData')
      if (outData.report) {
        reporterQueue.add('aggregator', outData, { removeOnComplete: true })
      }
    } catch (e) {
      logger.error(e)
    } finally {
      heartbeatQueue.add('random-heartbeat', inData, {
        delay: uniform(0, inData.randomHeartbeatRate.value),
        removeOnComplete: true
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
async function prepareDataForReporter(
  data: IAggregatorHeartbeatWorker,
  _logger: Logger
): Promise<IAggregatorWorkerReporter> {
  const logger = _logger.child({ name: 'prepareDataForReporter', file: FILE_NAME })

  const callbackAddress = data.address
  const submission = await fetchDataWithAdapter(data.adapter)
  let report = data.report

  const oracleRoundState = await oracleRoundStateCall(data.address, ORACLE_ADDRESS, _logger)
  logger.debug(oracleRoundState, 'oracleRoundState')
  const lastSubmission = oracleRoundState._latestSubmission.toNumber()
  if (report === undefined) {
    report = shouldReport(lastSubmission, submission, data.threshold, data.absoluteThreshold)
    logger.debug(report, 'report')
  }

  return {
    report,
    callbackAddress,
    roundId: oracleRoundState._roundId,
    submission
  }
}

function shouldReport(
  lastSubmission: number,
  submission: number,
  threshold: number,
  absoluteThreshold: number
): boolean {
  if (lastSubmission && submission) {
    const range = lastSubmission * threshold
    const l = lastSubmission - range
    const r = lastSubmission + range
    return submission < l || submission > r
  } else if (!lastSubmission && submission) {
    // lastSubmission hit zero
    return submission > absoluteThreshold
  }

  // Something strange happened, don't report!
  return false
}

async function oracleRoundStateCall(
  aggregatorAddress: string,
  oracleAddress: string,
  logger: Logger
): Promise<IOracleRoundState> {
  logger.debug({ name: 'oracleRoundStateCall', file: FILE_NAME })

  const provider = new ethers.providers.JsonRpcProvider(PROVIDER_URL)
  const aggregator = new ethers.Contract(aggregatorAddress, Aggregator__factory.abi, provider)
  return await aggregator.oracleRoundState(oracleAddress, 0)
}

function addReportProperty(o, report: boolean) {
  return Object.assign({}, ...[o, { report }])
}
