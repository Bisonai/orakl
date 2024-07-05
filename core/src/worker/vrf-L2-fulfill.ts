import { L2Endpoint__factory } from '@bisonai/orakl-contracts/v0.1'
import { Queue, Worker } from 'bullmq'
import { BigNumber, ethers } from 'ethers'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import {
  BULLMQ_CONNECTION,
  L2_REPORTER_VRF_FULFILL_QUEUE_NAME,
  L2_WORKER_VRF_FULFILL_QUEUE_NAME,
  VRF_FULFILL_GAS_MINIMUM,
  VRF_FULLFILL_GAS_PER_WORD,
  WORKER_JOB_SETTINGS,
} from '../settings'
import {
  IL2VrfFulfillListenerWorker,
  IL2VrfFulfillTransactionParameters,
  ITransactionParameters,
  QueueType,
} from '../types'

const FILE_NAME = import.meta.url

export async function worker(redisClient: RedisClientType, _logger: Logger) {
  const logger = _logger.child({ name: 'worker', file: FILE_NAME })
  const queue = new Queue(L2_REPORTER_VRF_FULFILL_QUEUE_NAME, BULLMQ_CONNECTION)
  const worker = new Worker(
    L2_WORKER_VRF_FULFILL_QUEUE_NAME,
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
    const inData: IL2VrfFulfillListenerWorker = job.data
    logger.debug(inData, 'inData')

    try {
      const payloadParameters: IL2VrfFulfillTransactionParameters = {
        requestId: inData.l2RequestId,
        randomWords: inData.randomWords,
        callbackGasLimit: BigNumber.from(inData.callbackGasLimit).toNumber(),
      }
      const to = inData.callbackAddress
      const tx = buildTransaction(
        payloadParameters,
        to,
        VRF_FULFILL_GAS_MINIMUM + VRF_FULLFILL_GAS_PER_WORD * inData.randomWords.length,
        iface,
        logger,
      )
      logger.debug(tx, 'tx')

      await reporterQueue.add('vrf', tx, {
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
  payloadParameters: IL2VrfFulfillTransactionParameters,
  to: string,
  gasMinimum: number,
  iface: ethers.utils.Interface,
  logger: Logger,
): ITransactionParameters {
  const { requestId, randomWords, callbackGasLimit } = payloadParameters
  const gasLimit = callbackGasLimit + gasMinimum
  const payload = iface.encodeFunctionData('fulfillRandomWords', [requestId, randomWords])
  logger.debug(payload, 'payload')
  return {
    payload,
    gasLimit,
    to,
  }
}
