import { Worker, Queue } from 'bullmq'
import axios from 'axios'
import { Logger } from 'pino'
import { loadAdapters } from './utils'
import { IPredefinedFeedListenerWorker, IPredefinedFeedWorkerReporter } from '../types'
import { pipe } from '../utils'
import {
  localAggregatorFn,
  WORKER_PREDEFINED_FEED_QUEUE_NAME,
  REPORTER_PREDEFINED_FEED_QUEUE_NAME,
  BULLMQ_CONNECTION
} from '../settings'

const FILE_NAME = import.meta.url

// FIXME currently not used!

export async function predefinedFeedWorker(_logger: Logger) {
  const logger = _logger.child({ name: 'predefinedFeedWorker', file: FILE_NAME })

  const adapters = (await loadAdapters({ postprocess: true }))[0] // FIXME take all adapters
  logger.debug(adapters, 'adapters')

  new Worker(
    WORKER_PREDEFINED_FEED_QUEUE_NAME,
    predefinedFeedJob(REPORTER_PREDEFINED_FEED_QUEUE_NAME, adapters),
    BULLMQ_CONNECTION
  )
}

function predefinedFeedJob(queueName, adapters, _logger?: Logger) {
  const logger = _logger?.child({ name: 'predefinedFeedJob', file: FILE_NAME })
  const queue = new Queue(queueName, BULLMQ_CONNECTION)

  async function wrapper(job) {
    const inData: IPredefinedFeedListenerWorker = job.data
    logger?.debug(inData, 'inData')

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
            logger?.error(e)
          }
        })
      )
      logger?.debug(allResults, 'allResults')

      // FIXME single node aggregation of multiple results
      // FIXME check if aggregator is defined and if exists
      const res = localAggregatorFn(...allResults)
      logger?.debug(res, 'res')

      const outData: IPredefinedFeedWorkerReporter = {
        requestId: inData.requestId,
        jobId: inData.jobId,
        callbackAddress: inData.callbackAddress,
        callbackFunctionId: inData.callbackFunctionId,
        data: res
      }
      logger?.debug(outData, 'outData')

      await queue.add('predefined-feed', outData)
    } catch (e) {
      logger?.error(e)
    }
  }

  return wrapper
}
