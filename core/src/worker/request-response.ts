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
  IRequestResponseListenerWorker,
  IRequestResponseTransactionParameters,
  RequestCommitmentRequestResponse,
  QueueType,
  ITransactionParameters
} from '../types'
import { pipe } from '../utils'
import {
  WORKER_REQUEST_RESPONSE_QUEUE_NAME,
  REPORTER_REQUEST_RESPONSE_QUEUE_NAME,
  BULLMQ_CONNECTION,
  WORKER_JOB_SETTINGS,
  REQUEST_RESPONSE_FULFILL_GAS_MINIMUM
} from '../settings'
import {
  JOB_ID_MAPPING,
  JOB_ID_UINT128,
  JOB_ID_INT256,
  JOB_ID_BOOL,
  JOB_ID_STRING,
  JOB_ID_BYTES32,
  JOB_ID_BYTES
} from './request-response.utils'

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

export async function job(queue: QueueType, _logger: Logger) {
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

      await queue.add('request-response', tx, {
        jobId: inData.requestId,
        ...WORKER_JOB_SETTINGS
      })

      return tx
    } catch (e) {
      logger.error(e)
      throw e
    }
  }

  return wrapper
}

function buildTransaction(
  payloadParameters: IRequestResponseTransactionParameters,
  to: string,
  gasMinimum: number,
  iface: ethers.utils.Interface,
  _logger: Logger
): ITransactionParameters {
  const gasLimit = payloadParameters.callbackGasLimit + gasMinimum

  const fulfillDataRequestFn = JOB_ID_MAPPING[payloadParameters.jobId]
  if (fulfillDataRequestFn == undefined) {
    throw new Error() // FIXME
  }

  let response
  switch (payloadParameters.jobId) {
    case JOB_ID_UINT128:
    case JOB_ID_INT256:
      response = Math.floor(payloadParameters.response)
      break
    case JOB_ID_BOOL:
      if (payloadParameters.response.toLowerCase() == 'false') {
        response = false
      } else {
        response = Boolean(payloadParameters.response)
      }
      break
    case JOB_ID_STRING:
      response = String(payloadParameters.response)
      break
    case JOB_ID_BYTES32:
    case JOB_ID_BYTES:
      response = payloadParameters.response
      break
  }

  const rc: RequestCommitmentRequestResponse = [
    payloadParameters.blockNum,
    payloadParameters.accId,
    payloadParameters.numSubmission,
    payloadParameters.callbackGasLimit,
    payloadParameters.sender
  ]

  const payload = iface.encodeFunctionData(fulfillDataRequestFn, [
    payloadParameters.requestId,
    response,
    rc,
    payloadParameters.isDirectPayment
  ])

  return {
    payload,
    gasLimit,
    to
  }
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
