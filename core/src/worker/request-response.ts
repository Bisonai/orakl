import { ethers } from 'ethers'
import { Worker, Queue } from 'bullmq'
import axios from 'axios'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { RequestResponseCoordinator__factory } from '@bisonai/orakl-contracts'
import { buildReducer } from './utils'
import { decodeRequest } from './decoding'
import { requestResponseReducerMapping } from './reducer'
import {
  IErrorMsgData,
  IRequestResponseListenerWorker,
  IRequestResponseTransactionParameters,
  QueueType
} from '../types'
import { pipe } from '../utils'
import { buildTransaction } from './request-response.utils'
import {
  WORKER_REQUEST_RESPONSE_QUEUE_NAME,
  REPORTER_REQUEST_RESPONSE_QUEUE_NAME,
  BULLMQ_CONNECTION,
  WORKER_JOB_SETTINGS,
  REQUEST_RESPONSE_FULFILL_GAS_MINIMUM
} from '../settings'
import { storeErrorMsg } from './api'

const FILE_NAME = import.meta.url

export async function worker(redisClient: RedisClientType, _logger: Logger) {
  const logger = _logger.child({ name: 'worker', file: FILE_NAME })
  const queue = new Queue(REPORTER_REQUEST_RESPONSE_QUEUE_NAME, BULLMQ_CONNECTION)
  const worker = new Worker(
    WORKER_REQUEST_RESPONSE_QUEUE_NAME,
    await job(queue, _logger),
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

export async function job(reporterQueue: QueueType, _logger: Logger) {
  const logger = _logger.child({ name: 'job', file: FILE_NAME })
  const iface = new ethers.utils.Interface(RequestResponseCoordinator__factory.abi)

  async function wrapper(job) {
    const inData: IRequestResponseListenerWorker = job.data
    logger.debug(inData, 'inData')

    try {
      const response = await processRequest(inData.data, _logger)

      const payloadParameters: IRequestResponseTransactionParameters = {
        blockNum: inData.blockNum,
        accId: inData.accId,
        jobId: inData.jobId,
        requestId: inData.requestId,
        numSubmission: inData.numSubmission,
        callbackGasLimit: inData.callbackGasLimit,
        sender: inData.sender,
        isDirectPayment: inData.isDirectPayment,
        response
      }
      const to = inData.callbackAddress

      const tx = buildTransaction(
        payloadParameters,
        to,
        REQUEST_RESPONSE_FULFILL_GAS_MINIMUM,
        iface,
        logger
      )
      logger.debug(tx, 'tx')

      await reporterQueue.add('request-response', tx, {
        jobId: inData.requestId,
        ...WORKER_JOB_SETTINGS
      })

      return tx
    } catch (e) {
      logger.error(e)
      const errorMsg = {
        type: e.type,
        message: e.message,
        stack: e.stack,
        code: e.code,
        name: e.name
      }

      const errorData: IErrorMsgData = {
        requestId: inData.requestId,
        timestamp: new Date(Date.now()),
        errorMsg: JSON.stringify(errorMsg)
      }

      await storeErrorMsg({ data: errorData, logger: _logger })

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
