import { ethers } from 'ethers'
import { Worker, Queue } from 'bullmq'
import {
  fetchDataWithAdapter,
  loadAdapters,
  loadAggregators,
  mergeAggregatorsAdapters,
  uniform,
  addReportProperty
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
import { PROVIDER_URL, PUBLIC_KEY as ORACLE_ADDRESS } from '../load-parameters'
import { Aggregator__factory } from '@bisonai-cic/icn-contracts'

export async function aggregatorWorker() {
  console.debug('aggregatorWorker')

  const adapters = await loadAdapters()
  console.debug('aggregatorWorker:adapters', adapters)

  const aggregators = await loadAggregators()
  console.debug('aggregatorWorker:aggregators', aggregators)

  const aggregatorsWithAdapters = mergeAggregatorsAdapters(aggregators, adapters)
  console.debug('aggregatorWorker:aggregatorsWithAdapters', aggregatorsWithAdapters)

  // Event based worker
  new Worker(
    WORKER_AGGREGATOR_QUEUE_NAME,
    aggregatorJob(REPORTER_AGGREGATOR_QUEUE_NAME, aggregatorsWithAdapters),
    BULLMQ_CONNECTION
  )

  // Fixed heartbeat worker
  new Worker(
    FIXED_HEARTBEAT_QUEUE_NAME,
    fixedHeartbeatJob(
      FIXED_HEARTBEAT_QUEUE_NAME,
      REPORTER_AGGREGATOR_QUEUE_NAME,
      aggregatorsWithAdapters
    ),
    BULLMQ_CONNECTION
  )

  // Random heartbeat worker
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

function aggregatorJob(reporterQueueName: string, aggregatorsWithAdapters) {
  // This job is coming from on-chain request (event NewRound). Oracle
  // needs to submit the latest data based on this request without any
  // check on time or data change.
  const reporterQueue = new Queue(reporterQueueName, BULLMQ_CONNECTION)

  async function wrapper(job) {
    const inData: IAggregatorListenerWorker = job.data
    console.debug('aggregatorJob:inData', inData)

    const ag = addReportProperty(aggregatorsWithAdapters[inData.aggregatorAddress], true)

    try {
      const outData = await prepareDataForReporter(ag)
      console.debug('aggregatorJob:outData', outData)
      reporterQueue.add('aggregator', outData)
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
  for (const k in agregatorsWithAdapters) {
    const ag = agregatorsWithAdapters[k]
    if (ag.fixedHeartbeatRate.active) {
      heartbeatQueue.add('fixed-heartbeat', addReportProperty(ag, true), {
        delay: ag.fixedHeartbeatRate.value
      })
    }
  }

  async function wrapper(job) {
    const inData: IAggregatorHeartbeatWorker = job.data
    console.debug('fixedHeartbeatJob:inData', inData)
    const outData = await prepareDataForReporter(inData)
    console.debug('fixedHeartbeatJob:outData', outData)

    reporterQueue.add('aggregator', outData)
    heartbeatQueue.add('fixed-heartbeat', inData, { delay: inData.fixedHeartbeatRate.value })
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
  for (const k in agregatorsWithAdapters) {
    const ag = agregatorsWithAdapters[k]
    if (ag.randomHeartbeatRate.active) {
      heartbeatQueue.add('random-heartbeat', ag, {
        delay: uniform(0, ag.randomHeartbeatRate.value)
      })
    }
  }

  async function wrapper(job) {
    const inData: IAggregatorHeartbeatWorker = job.data
    console.debug('randomHeartbeatJob:inData', inData)
    const outData = await prepareDataForReporter(inData)
    console.debug('randomHeartbeatJob:outData', outData)

    if (outData.report) {
      reporterQueue.add('aggregator', outData)
    }
    heartbeatQueue.add('random-heartbeat', inData, {
      delay: uniform(0, inData.randomHeartbeatRate.value)
    })
  }

  return wrapper
}

async function prepareDataForReporter(data): Promise<IAggregatorWorkerReporter> {
  const callbackAddress = data.aggregatorAddress
  const submission = await fetchDataWithAdapter(data.adapter)
  let report = data.report

  const oracleRoundState = await oracleRoundStateCall(data.aggregatorAddress, ORACLE_ADDRESS)
  console.debug('prepareDataForReporter:oracleRoundState', oracleRoundState)
  const lastSubmission = oracleRoundState._latestSubmission.toNumber()
  if (report === undefined) {
    report = shouldReport(lastSubmission, submission, data.threshold, data.absoluteThreshold)
    console.log('prepareDataForReporter:report', report)
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
  oracleAddress: string
): Promise<IOracleRoundState> {
  console.debug('oracleRoundStateCall')
  const provider = new ethers.providers.JsonRpcProvider(PROVIDER_URL)
  const aggregator = new ethers.Contract(aggregatorAddress, Aggregator__factory.abi, provider)
  return await aggregator.oracleRoundState(oracleAddress, 0)
}
