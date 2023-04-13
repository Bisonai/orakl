import { Worker, Queue } from 'bullmq'
import axios from 'axios'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { buildReducer } from './utils'
import { decodeRequest } from './decoding'
import { requestResponseReducerMapping } from './reducer'
import { IRequestResponseListenerWorker, IRequestResponseWorkerReporter } from '../types'
import { pipe } from '../utils'
import {
  WORKER_REQUEST_RESPONSE_QUEUE_NAME,
  REPORTER_REQUEST_RESPONSE_QUEUE_NAME,
  BULLMQ_CONNECTION
} from '../settings'

const FILE_NAME = import.meta.url

export async function worker(redisClient: RedisClientType, _logger: Logger) {
  const logger = _logger.child({ name: 'worker', file: FILE_NAME })
  const worker = new Worker(
    WORKER_REQUEST_RESPONSE_QUEUE_NAME,
    job(REPORTER_REQUEST_RESPONSE_QUEUE_NAME, _logger),
    BULLMQ_CONNECTION
  )

  async function handleExit() {
    logger.info('Exiting. Wait for graceful shutdown.')

    await redisClient.quit()
    await worker.close()
  }
  process.on('SIGINT', handleExit)
  process.on('SIGTERM', handleExit)
}

function job(queueName: string, _logger: Logger) {
  const queue = new Queue(queueName, BULLMQ_CONNECTION)
  const logger = _logger.child({ name: 'job', file: FILE_NAME })

  async function wrapper(job) {
    const inData: IRequestResponseListenerWorker = job.data
    logger.debug(inData, 'inData')

    try {
      const res = await processRequest(inData.data, _logger)

      const outData: IRequestResponseWorkerReporter = {
        callbackAddress: inData.callbackAddress,
        blockNum: inData.blockNum,
        requestId: inData.requestId,
        jobId: inData.jobId,
        accId: inData.accId,
        callbackGasLimit: inData.callbackGasLimit,
        sender: inData.sender,
        isDirectPayment: inData.isDirectPayment,
        data: res
      }
      logger.debug(outData, 'outData')

      await queue.add('request-response', outData)
    } catch (e) {
      logger.error(e)
      throw e
    }
  }

  return wrapper
}

async function processRequest(reqEnc: string, _logger: Logger): Promise<string | number> {
  const logger = _logger.child({ name: 'processRequest', file: FILE_NAME })
  const req = await decodeRequest(reqEnc)
  logger.debug(req, 'req')

  const options = {
    method: 'GET'
  }
  const rawData = (await axios.get(req[0].args, options)).data
  const reducers = buildReducer(requestResponseReducerMapping, req.slice(1))
  const res = pipe(...reducers)(rawData)

  logger.debug(res, 'res')
  return res
}
