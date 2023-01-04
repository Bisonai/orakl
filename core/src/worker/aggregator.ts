import * as Fs from 'node:fs/promises'
import * as Path from 'node:path'
import { ethers } from 'ethers'
import { Worker, Queue } from 'bullmq'
import {
  fetchDataWithAdapter,
  loadAdapters,
  loadAggregators,
  mergeAggregatorsAdapters
} from './utils'
import { reducerMapping } from './reducer'
import {
  IAggregatorListenerWorker,
  IAggregatorWorkerReporter,
  IAggregatorFixedHeartbeatWorker
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

  // const res = adapters['0xf84be3681d32250d9fbe85ab1c56db59d9e1ef2dff242de066853b4a047e15e2'][0]
  // const res = aggregatorsWithAdapters[0].adapter[0].reducers

  // console.log(res)
  // process.exit(0)

  // console.log(adapters['0x47c99abed3324a2707c28affff1267e45918ec8c3f20b8aa892e8b065d2942dd'])
  // console.log(
  //   adapters['0x47c99abed3324a2707c28affff1267e45918ec8c3f20b8aa892e8b065d2942dd'][0]['reducers'][0]
  // )
  // process.exit(0)

  // console.log(adapters['0xf84be3681d32250d9fbe85ab1c56db59d9e1ef2dff242de066853b4a047e15e2'])

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
  // new Worker(
  //   RANDOM_HEARTBEAT_QUEUE_NAME,
  //   randomHeartbeatJob(
  //     RANDOM_HEARTBEAT_QUEUE_NAME,
  //     REPORTER_AGGREGATOR_QUEUE_NAME,
  //     aggregatorsWithAdapters
  //   ),
  //   BULLMQ_CONNECTION
  // )
}

function aggregatorJob(queueName, adapters) {
  const queue = new Queue(queueName, BULLMQ_CONNECTION)

  async function wrapper(job) {
    const inData: IAggregatorListenerWorker = job.data
    console.debug('aggregatorJob:inData', inData)

    try {
      // TODO Fetch data (same as in Predefined Feed or Request-Response)
      const outData: IAggregatorWorkerReporter = {
        callbackAddress: '0x',
        roundId: 0,
        submission: 0
      }
      console.debug('aggregatorJob:outData', outData)

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

      if (inData.mustReport || dataDiverged) {
        await queue.add('aggregator', outData)
      }
    } catch (e) {
      console.error(e)
    }
  }

  return wrapper
}

function fixedHeartbeatJob(
  heartbeatQueueName: string,
  reporterQueueName: string,
  agregatorsWithAdapters: IAggregatorFixedHeartbeatWorker[]
) {
  console.debug('fixedHeartbeatJob')

  const heartbeatQueue = new Queue(heartbeatQueueName, BULLMQ_CONNECTION)
  const reporterQueue = new Queue(reporterQueueName, BULLMQ_CONNECTION)

  for (const ag of agregatorsWithAdapters) {
    heartbeatQueue.add('fixed-heartbeat', ag, { delay: ag.fixedHeartbeatRate })
  }

  async function wrapper(job) {
    const inData: IAggregatorFixedHeartbeatWorker = job.data
    const outData = await prepareDataForReporter(inData)
    reporterQueue.add('aggregator', outData)
    // heartbeatQueue.add('fixed-heartbeat', inData, { delay: inData.fixedHeartbeatRate })
  }

  return wrapper
}

async function prepareDataForReporter(data): Promise<IAggregatorWorkerReporter> {
  const callbackAddress = data.aggregatorAddress
  const submission = await fetchDataWithAdapter(data.adapter)

  let roundId

  try {
    const lastSubmission = await latestRoundData(data.aggregatorAddress)
    console.debug('fixedHeartbeatJob:lastSubmission', lastSubmission)
    roundId = undefined // TODO extract roundId from the last submission
  } catch (e) {
    if (e.code == 'CALL_EXCEPTION' && e.reason == 'No data present') {
      // No data were submitted to feed yet! Submitting for the
      // first time!
      roundId = 1
    }
  }

  return {
    callbackAddress,
    roundId: 1,
    submission: submission
  }
}

async function latestRoundData(address: string) {
  const provider = new ethers.providers.JsonRpcProvider(PROVIDER_URL)
  const aggregator = new ethers.Contract(address, Aggregator__factory.abi, provider)
  return await aggregator.latestRoundData()
}
