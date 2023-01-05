import * as Fs from 'node:fs/promises'
import * as Path from 'node:path'
import { ethers, BigNumber } from 'ethers'
import { Worker, Queue } from 'bullmq'
import {
  fetchDataWithAdapter,
  loadAdapters,
  loadAggregators,
  mergeAggregatorsAdapters,
  uniform
} from './utils'
import { reducerMapping } from './reducer'
import {
  IAggregatorListenerWorker,
  IAggregatorWorkerReporter,
  IAggregatorHeartbeatWorker,
  ILatestRoundData
} from '../types'
import {
  WORKER_AGGREGATOR_QUEUE_NAME,
  REPORTER_AGGREGATOR_QUEUE_NAME,
  FIXED_HEARTBEAT_QUEUE_NAME,
  RANDOM_HEARTBEAT_QUEUE_NAME,
  BULLMQ_CONNECTION
} from '../settings'
import { PROVIDER_URL } from '../load-parameters'
import { Aggregator__factory } from '@bisonai-cic/icn-contracts'

export async function aggregatorWorker() {
  console.debug('aggregatorWorker')

  const adapters = await loadAdapters()
  console.debug('aggregatorWorker:adapters', adapters)

  const aggregators = await loadAggregators()
  console.debug('aggregatorWorker:aggregators', aggregators)

  const aggregatorsWithAdapters = mergeAggregatorsAdapters(aggregators, adapters)
  console.debug('aggregatorWorker:aggregatorsWithAdapters', aggregatorsWithAdapters)

  // new Worker(
  //   WORKER_AGGREGATOR_QUEUE_NAME,
  //   aggregatorJob(REPORTER_AGGREGATOR_QUEUE_NAME, adapters),
  //   BULLMQ_CONNECTION
  // )

  // Fixed Heartbeat
  new Worker(
    FIXED_HEARTBEAT_QUEUE_NAME,
    fixedHeartbeatJob(
      FIXED_HEARTBEAT_QUEUE_NAME,
      REPORTER_AGGREGATOR_QUEUE_NAME,
      aggregatorsWithAdapters
    ),
    BULLMQ_CONNECTION
  )

  // Random Heartbeat
  new Worker(
    RANDOM_HEARTBEAT_QUEUE_NAME,
    randomHeartbeatJob(
      RANDOM_HEARTBEAT_QUEUE_NAME,
      REPORTER_AGGREGATOR_QUEUE_NAME,
      aggregatorsWithAdapters
    ),
    BULLMQ_CONNECTION
  )
}

function aggregatorJob(reporterQueueName, adapters) {
  // This job is coming from on-chain request (event NewRound). Oracle
  // needs to submit the latest data based on this request without any
  // check of data change compared to previous submissions.
  const reporterQueue = new Queue(reporterQueueName, BULLMQ_CONNECTION)

  async function wrapper(job) {
    const inData: IAggregatorListenerWorker = job.data
    console.debug('aggregatorJob:inData', inData)

    try {
      // TODO Fetch data (same as in Predefined Feed or Request-Response)
      // const outData: IAggregatorWorkerReporter = {
      //   callbackAddress: '0x',
      //   roundId: 0,
      //   submission: 0
      // }
      // console.debug('aggregatorJob:outData', outData)

      let dataDiverged = false
      if (inData.mustReport) {
        // TODO check
      } else {
        // Check if the new value reaches over threshold or absoluteThreshold.
        if (dataDiverged) {
          dataDiverged = true
        } else {
          // TODO Put Fixed Heartbeat to appropriate queue
        }
      }

      // if (inData.mustReport || dataDiverged) {
      //   await reporterQueue.add('aggregator', outData)
      // }
    } catch (e) {
      console.error(e)
    }
  }

  return wrapper
}

function fixedHeartbeatJob(
  heartbeatQueueName: string,
  reporterQueueName: string,
  agregatorsWithAdapters: IAggregatorHeartbeatWorker[]
) {
  console.debug('fixedHeartbeatJob')

  const heartbeatQueue = new Queue(heartbeatQueueName, BULLMQ_CONNECTION)
  const reporterQueue = new Queue(reporterQueueName, BULLMQ_CONNECTION)

  // Launch all aggregators to be executed with fixed heartbeat
  // TODO Add clock synchronization through on-chain public data timestamp
  for (const ag of agregatorsWithAdapters) {
    heartbeatQueue.add('fixed-heartbeat', ag, { delay: ag.fixedHeartbeatRate })
  }

  async function wrapper(job) {
    const inData: IAggregatorHeartbeatWorker = job.data
    const outData = await prepareDataForReporter(inData, true)
    reporterQueue.add('aggregator', outData)
    heartbeatQueue.add('fixed-heartbeat', inData, { delay: inData.fixedHeartbeatRate })
  }

  return wrapper
}

function randomHeartbeatJob(
  heartbeatQueueName: string,
  reporterQueueName: string,
  agregatorsWithAdapters: IAggregatorHeartbeatWorker[]
) {
  console.debug('randomHeartbeatJob')

  const heartbeatQueue = new Queue(heartbeatQueueName, BULLMQ_CONNECTION)
  const reporterQueue = new Queue(reporterQueueName, BULLMQ_CONNECTION)

  // Launch all aggregators to be executed with random heartbeat
  for (const ag of agregatorsWithAdapters) {
    heartbeatQueue.add('random-heartbeat', ag, { delay: uniform(0, ag.randomHeartbeatRate) })
  }

  async function wrapper(job) {
    const inData: IAggregatorHeartbeatWorker = job.data
    const outData = await prepareDataForReporter(inData)
    if (outData.report) {
      reporterQueue.add('aggregator', outData)
    }
    heartbeatQueue.add('random-heartbeat', inData, {
      delay: uniform(0, inData.randomHeartbeatRate)
    })
  }

  return wrapper
}

async function prepareDataForReporter(data, _report?: boolean): Promise<IAggregatorWorkerReporter> {
  const callbackAddress = data.aggregatorAddress
  const submission = await fetchDataWithAdapter(data.adapter)

  let roundId = BigNumber.from(1)
  let report = _report

  try {
    const latestRoundData = await latestRoundDataCall(data.aggregatorAddress)
    const lastSubmission = latestRoundData.answer.toNumber()
    if (report === undefined) {
      report = shouldReport(lastSubmission, submission, data.threshold, data.absoluteThreshold)
      console.log('prepareDtaForReporter:report', report)
    }
    roundId = latestRoundData.roundId.add(1)
    console.debug('prepareDataForReporter:latestRoundData', latestRoundData)
  } catch (e) {
    if (e.code == 'CALL_EXCEPTION' && e.reason == 'No data present') {
      // No data were submitted to feed yet! Submitting for the
      // first time!
    }
  }

  return {
    report: report || false,
    callbackAddress,
    roundId,
    submission: submission
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

async function latestRoundDataCall(address: string): Promise<ILatestRoundData> {
  const provider = new ethers.providers.JsonRpcProvider(PROVIDER_URL)
  const aggregator = new ethers.Contract(address, Aggregator__factory.abi, provider)
  return await aggregator.latestRoundData()
}
