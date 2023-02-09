import { Worker, Queue } from 'bullmq'
import axios from 'axios'
import { Logger } from 'pino'
import { IRequestResponseListenerWorker, IRequestResponseWorkerReporter } from '../types'
import { readFromJson } from '../utils'
import {
  WORKER_REQUEST_RESPONSE_QUEUE_NAME,
  REPORTER_REQUEST_RESPONSE_QUEUE_NAME,
  BULLMQ_CONNECTION
} from '../settings'
import { decodeRequest } from './decoding'

const FILE_NAME = import.meta.url

export async function worker(_logger: Logger) {
  _logger.debug({ name: 'worker', file: FILE_NAME })
  new Worker(
    WORKER_REQUEST_RESPONSE_QUEUE_NAME,
    job(REPORTER_REQUEST_RESPONSE_QUEUE_NAME, _logger),
    BULLMQ_CONNECTION
  )
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
    }
  }

  return wrapper
}

async function processRequest(reqEnc: string, _logger: Logger): Promise<string | number> {
  const logger = _logger.child({ name: 'processRequest', file: FILE_NAME })
  const req = await decodeRequest(reqEnc)
  logger.debug(req, 'req')

  let res: string = (await axios.get(req.get)).data
  if (req.path) {
    res = readFromJson(res, req.path)
  }

  logger.debug(res, 'res')
  return res
}
