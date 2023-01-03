import * as Fs from 'node:fs/promises'
import * as Path from 'node:path'
import { ethers } from 'ethers'
import { Worker, Queue } from 'bullmq'
// import { got } from 'got'
import { loadAdapters, loadAggregators } from './utils'
import { reducerMapping } from './reducer'
import { IAggregatorListenerWorker, IAggregatorWorkerReporter } from '../types'
import {
  WORKER_AGGREGATOR_QUEUE_NAME,
  REPORTER_AGGREGATOR_QUEUE_NAME,
  BULLMQ_CONNECTION
} from '../settings'

export async function aggregatorWorker() {
  console.debug('aggregatorWorker')

  const adapters = await loadAdapters()
  console.debug('main:adapters', adapters)

  const aggregators = await loadAggregators()
  console.debug('main:aggregators', aggregators)

  new Worker(
    WORKER_AGGREGATOR_QUEUE_NAME,
    aggregatorJob(REPORTER_AGGREGATOR_QUEUE_NAME, adapters),
    BULLMQ_CONNECTION
  )

  // Fixed Heartbeat
  // Random Heardbeat
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
