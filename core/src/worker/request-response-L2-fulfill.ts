import { L2Endpoint__factory } from '@bisonai/orakl-contracts/v0.1'
import { Queue, Worker } from 'bullmq'
import { BigNumber, ethers } from 'ethers'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import {
  BULLMQ_CONNECTION,
  L2_ENDPOINT,
  L2_REPORTER_REQUEST_RESPONSE_FULFILL_QUEUE_NAME,
  L2_WORKER_REQUEST_RESPONSE_FULFILL_QUEUE_NAME,
  REQUEST_RESPONSE_FULFILL_GAS_MINIMUM,
  WORKER_JOB_SETTINGS,
} from '../settings'
import {
  IL2RequestResponseFulfillListenerWorker,
  IL2RequestResponseFulfillTransactionParameters,
  ITransactionParameters,
  QueueType,
} from '../types'
import { JOB_ID_MAPPING } from './request-response.utils'

const FILE_NAME = import.meta.url

export async function worker(redisClient: RedisClientType, _logger: Logger) {
  const logger = _logger.child({ name: 'worker', file: FILE_NAME })
  const queue = new Queue(L2_REPORTER_REQUEST_RESPONSE_FULFILL_QUEUE_NAME, BULLMQ_CONNECTION)
  const worker = new Worker(
    L2_WORKER_REQUEST_RESPONSE_FULFILL_QUEUE_NAME,
    await job(queue, _logger),
    BULLMQ_CONNECTION,
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
  const logger = _logger.child({ name: 'vrfJob', file: FILE_NAME })
  const iface = new ethers.utils.Interface(L2Endpoint__factory.abi)

  async function wrapper(job) {
    const inData: IL2RequestResponseFulfillListenerWorker = job.data
    logger.debug(inData, 'inData')

    try {
      const payloadParameters: IL2RequestResponseFulfillTransactionParameters = {
        requestId: inData.l2RequestId,
        response: inData.response,
        callbackGasLimit: BigNumber.from(inData.callbackGasLimit).toNumber(),
        jobId: inData.jobId,
      }
      const to = L2_ENDPOINT
      const tx = buildTransaction(
        payloadParameters,
        to,
        REQUEST_RESPONSE_FULFILL_GAS_MINIMUM,
        iface,
        logger,
      )
      logger.debug(tx, 'tx')

      await reporterQueue.add('l2RRFulfill', tx, {
        jobId: inData.requestId,
        ...WORKER_JOB_SETTINGS,
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
  payloadParameters: IL2RequestResponseFulfillTransactionParameters,
  to: string,
  gasMinimum: number,
  iface: ethers.utils.Interface,
  logger: Logger,
): ITransactionParameters {
  const { requestId, response, callbackGasLimit, jobId } = payloadParameters
  const gasLimit = callbackGasLimit + gasMinimum
  const fulfillFn = JOB_ID_MAPPING[jobId]
  const payload = iface.encodeFunctionData(fulfillFn, [requestId, response])
  logger.debug(payload, 'payload')
  return {
    payload,
    gasLimit,
    to,
  }
}
