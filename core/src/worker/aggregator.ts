import { Worker, Queue, Job } from 'bullmq'
import { Logger } from 'pino'
import {
  getAggregatorGivenAddress,
  getAggregator,
  getActiveAggregators,
  fetchDataFeed
} from './api'
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
  // RANDOM_HEARTBEAT_QUEUE_NAME,
  BULLMQ_CONNECTION,
  PUBLIC_KEY as OPERATOR_ADDRESS,
  DEPLOYMENT_NAME,
  REMOVE_ON_COMPLETE,
  REMOVE_ON_FAIL,
  CHAIN
} from '../settings'
import { IcnError, IcnErrorCode } from '../errors'
import { buildReporterJobId } from '../utils'
import {
  loadAdapters,
  loadAggregators,
  mergeAggregatorsAdapters,
  uniform,
  oracleRoundStateCall
} from './utils'

const FILE_NAME = import.meta.url

export async function aggregatorWorker(_logger: Logger) {
  const logger = _logger.child({ name: 'aggregatorWorker', file: FILE_NAME })

  const aggregators = await getActiveAggregators({ chain: CHAIN, logger })
  console.log(aggregators)

  const fixedHeartbeatQueue = new Queue(FIXED_HEARTBEAT_QUEUE_NAME, BULLMQ_CONNECTION)

  // Launch all active aggregators
  for (const ag of aggregators) {
    const aggregatorAddress = '0x0DCd1Bf9A1b36cE34237eEaFef220932846BCD82' // FIXME
    // const aggregatorAddress = ag['address'] // FIXME define type

    await fixedHeartbeatQueue.add(
      'heartbeat',
      { aggregatorAddress },
      {
        delay: await getSynchronizedDelay(aggregatorAddress, ag['heartbeat'], _logger), // FIXME define type
        removeOnComplete: true,
        removeOnFail: true
      }
    )

    // if (aggregator.randomHeartbeatRate.active) {
    //   await randomHeartbeatQueue.add('random-heartbeat', addReportProperty(aggregator, undefined), {
    //     delay: uniform(0, aggregator.randomHeartbeatRate.value),
    //     removeOnComplete: REMOVE_ON_COMPLETE,
    //     removeOnFail: REMOVE_ON_FAIL
    //   })
    // }
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

  // Random heartbeat worker
  // new Worker(
  //   RANDOM_HEARTBEAT_QUEUE_NAME,
  //   randomHeartbeatJob(
  //     RANDOM_HEARTBEAT_QUEUE_NAME,
  //     REPORTER_AGGREGATOR_QUEUE_NAME,
  //     aggregatorsWithAdapters,
  //     _logger
  //   ),
  //   BULLMQ_CONNECTION
  // )
}

function aggregatorJob(reporterQueueName: string, _logger: Logger) {
  const logger = _logger.child({ name: 'aggregatorJob', file: FILE_NAME })
  const reporterQueue = new Queue(reporterQueueName, BULLMQ_CONNECTION)

  async function wrapper(job: Job) {
    const inData: IAggregatorWorker = job.data
    logger.debug(inData, 'inData-regular')
    const aggregatorAddress = inData.aggregatorAddress
    const roundId = inData.roundId

    try {
      const { aggregatorHash, heartbeat } = await getAggregatorGivenAddress({
        aggregatorAddress,
        logger
      })

      const outData = await prepareDataForReporter({
        aggregatorHash,
        aggregatorAddress,
        report: true,
        workerSource: inData.workerSource,
        delay: heartbeat,
        roundId,
        _logger
      })

      logger.debug(outData, 'outData-regular')

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
      const oracleRoundState = await oracleRoundStateCall({
        aggregatorAddress,
        operatorAddress: OPERATOR_ADDRESS,
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
 * @deprecated The method should not be used
 */
// function randomHeartbeatJob(
//   heartbeatQueueName: string,
//   reporterQueueName: string,
//   aggregatorsWithAdapters: IAggregatorJob[],
//   _logger: Logger
// ) {
//   const logger = _logger.child({ name: 'randomHeartbeatJob', file: FILE_NAME })
//
//   const heartbeatQueue = new Queue(heartbeatQueueName, BULLMQ_CONNECTION)
//   const reporterQueue = new Queue(reporterQueueName, BULLMQ_CONNECTION)
//
//   async function wrapper(job: Job) {
//     const inData: IAggregatorJob = job.data
//     logger.debug(inData, 'inData-random')
//
//     const aggregatorAddress = inData.address
//     const aggregator = aggregatorsWithAdapters[aggregatorAddress]
//
//     if (!aggregator) {
//       throw new IcnError(IcnErrorCode.UndefinedAggregator)
//     }
//
//     try {
//       const outData = await prepareDataForReporter({
//         id: inData.id,
//         workerSource: 'random',
//         delay: aggregator.fixedHeartbeatRate.value, // FIXME
//         _logger
//       })
//       logger.debug(outData, 'outData-random')
//       if (outData.report) {
//         await reporterQueue.add('random', outData, {
//           removeOnComplete: REMOVE_ON_COMPLETE,
//           removeOnFail: REMOVE_ON_FAIL,
//           jobId: buildReporterJobId({
//             aggregatorAddress,
//             deploymentName: DEPLOYMENT_NAME,
//             roundId: outData.roundId
//           })
//         })
//       }
//     } catch (e) {
//       // It is possible that `FailedToFetchFromDataFeed` is raised from
//       // `prepareDataForReporter` which means that fetched data are
//       // not qualified to be used for submission. This exception is
//       // okay to ignore within random heartbeat because random
//       // heartbeat is executed in frequent intervals and data
//       // request will be performed soon again.
//       logger.error(e)
//     } finally {
//       await heartbeatQueue.add('random-heartbeat', inData, {
//         delay: uniform(0, inData.randomHeartbeatRate.value),
//         removeOnComplete: REMOVE_ON_COMPLETE,
//         removeOnFail: REMOVE_ON_FAIL
//       })
//     }
//   }
//
//   return wrapper
// }

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
  report,
  workerSource,
  delay,
  roundId,
  _logger
}: {
  aggregatorHash: string
  aggregatorAddress: string
  report?: boolean
  workerSource: string
  delay: number
  roundId?: number
  _logger: Logger
}): Promise<IAggregatorWorkerReporter> {
  const logger = _logger.child({ name: 'prepareDataForReporter', file: FILE_NAME })

  const { value } = await fetchDataFeed({ aggregatorHash, logger })

  const oracleRoundState = await oracleRoundStateCall({
    aggregatorAddress,
    operatorAddress: OPERATOR_ADDRESS,
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
 * Test whether the current submission deviates from the last
 * submission more than given threshold or absolute threshold. If yes,
 * return `true`, otherwise `false`.
 *
 * @param {number} latest submission value
 * @param {number} current submission value
 * @param {number} threshold configuration
 * @param {number} absolute threshold configuration
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

/**
 * Compute the number of seconds until the next round.
 *
 * @param {string} aggregator address
 * @param {number} heartbeat
 * @param {Logger}
 * @return {number} delay in seconds until the next round
 */
async function getSynchronizedDelay(
  aggregatorAddress: string,
  heartbeat: number,
  _logger: Logger
): Promise<number> {
  // FIXME modify aggregator to use single contract call

  let startedAt = 0
  const { _startedAt, _roundId } = await oracleRoundStateCall({
    aggregatorAddress,
    operatorAddress: OPERATOR_ADDRESS
  })

  if (_startedAt.toNumber() != 0) {
    startedAt = _startedAt.toNumber()
  } else {
    const { _startedAt } = await oracleRoundStateCall({
      aggregatorAddress,
      operatorAddress: OPERATOR_ADDRESS,
      roundId: Math.max(0, _roundId - 1)
    })
    startedAt = _startedAt.toNumber()
  }

  _logger.debug({ startedAt }, 'startedAt')
  const delay = heartbeat - (startedAt % heartbeat)
  _logger.debug({ delay }, 'delay')

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
