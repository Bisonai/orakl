import { Worker, Queue } from 'bullmq'
import axios from 'axios'
import { loadAdapters } from './utils'
import { IPredefinedFeedListenerWorker, IPredefinedFeedWorkerReporter } from '../types'
import { pipe } from '../utils'
import {
  localAggregatorFn,
  WORKER_PREDEFINED_FEED_QUEUE_NAME,
  REPORTER_PREDEFINED_FEED_QUEUE_NAME,
  BULLMQ_CONNECTION
} from '../settings'

export async function predefinedFeedWorker() {
  console.debug('predefinedFeedWorker')

  const adapters = (await loadAdapters({ postprocess: true }))[0] // FIXME take all adapters
  console.debug('predefinedFeedWorker:adapters', adapters)

  new Worker(
    WORKER_PREDEFINED_FEED_QUEUE_NAME,
    predefinedFeedJob(REPORTER_PREDEFINED_FEED_QUEUE_NAME, adapters),
    BULLMQ_CONNECTION
  )
}

function predefinedFeedJob(queueName, adapters) {
  const queue = new Queue(queueName, BULLMQ_CONNECTION)

  async function wrapper(job) {
    const inData: IPredefinedFeedListenerWorker = job.data
    console.debug('predefinedFeedJob:inData', inData)

    // FIXME take adapterId from job.data (information emitted by on-chain event)
    try {
      const allResults = await Promise.all(
        adapters[inData.jobId].map(async (adapter) => {
          const options = {
            method: adapter.method,
            headers: adapter.headers
          }

          try {
            const rawData = (await axios.get(adapter.url, options)).data
            return pipe(...adapter.reducers)(rawData)
          } catch (e) {
            console.error(e)
          }
        })
      )
      console.debug('predefinedFeedJob:allResults', allResults)

      // FIXME single node aggregation of multiple results
      // FIXME check if aggregator is defined and if exists
      const res = localAggregatorFn(...allResults)
      console.debug('predefinedFeedJob:res', res)

      const outData: IPredefinedFeedWorkerReporter = {
        requestId: inData.requestId,
        jobId: inData.jobId,
        callbackAddress: inData.callbackAddress,
        callbackFunctionId: inData.callbackFunctionId,
        data: res
      }
      console.debug('predefinedFeedJob:outData', outData)

      await queue.add('predefined-feed', outData)
    } catch (e) {
      console.error(e)
    }
  }

  return wrapper
}
