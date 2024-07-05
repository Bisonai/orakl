import { L1Endpoint__factory } from '@bisonai/orakl-contracts/v0.1'
import { Queue, Worker } from 'bullmq'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { getVrfConfig } from '../api'
import {
  BULLMQ_CONNECTION,
  CHAIN,
  L2_REPORTER_VRF_REQUEST_QUEUE_NAME,
  L2_WORKER_VRF_REQUEST_QUEUE_NAME,
  VRF_FULFILL_GAS_MINIMUM,
  WORKER_JOB_SETTINGS,
} from '../settings'
import {
  IL2EndpointListenerWorker,
  IL2VrfRequestTransactionParameters,
  ITransactionParameters,
  IVrfConfig,
  QueueType,
} from '../types'

const FILE_NAME = import.meta.url

export async function worker(redisClient: RedisClientType, _logger: Logger) {
  const logger = _logger.child({ name: 'worker', file: FILE_NAME })
  const queue = new Queue(L2_REPORTER_VRF_REQUEST_QUEUE_NAME, BULLMQ_CONNECTION)
  // FIXME add checks if exists and if includes all information
  const vrfConfig = await getVrfConfig({ chain: CHAIN, logger })
  const worker = new Worker(
    L2_WORKER_VRF_REQUEST_QUEUE_NAME,
    await job(queue, vrfConfig, _logger),
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

export async function job(reporterQueue: QueueType, config: IVrfConfig, _logger: Logger) {
  const logger = _logger.child({ name: 'vrfJob', file: FILE_NAME })
  const iface = new ethers.utils.Interface(L1Endpoint__factory.abi)

  async function wrapper(job) {
    const inData: IL2EndpointListenerWorker = job.data
    logger.debug(inData, 'inData')

    try {
      const payloadParameters: IL2VrfRequestTransactionParameters = {
        keyHash: inData.keyHash,
        blockNum: inData.blockNum,
        seed: inData.seed,
        accId: inData.accId,
        callbackGasLimit: inData.callbackGasLimit,
        numWords: inData.numWords,
        sender: inData.sender,
        l2RequestId: inData.requestId,
      }

      const to = inData.callbackAddress
      const tx = buildTransaction(payloadParameters, to, VRF_FULFILL_GAS_MINIMUM, iface, logger)
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
  payloadParameters: IL2VrfRequestTransactionParameters,
  to: string,
  gasMinimum: number,
  iface: ethers.utils.Interface,
  logger: Logger,
): ITransactionParameters {
  const { callbackGasLimit, keyHash, numWords, accId, sender, l2RequestId } = payloadParameters
  const gasLimit = callbackGasLimit + gasMinimum

  const payload = iface.encodeFunctionData('requestRandomWords', [
    keyHash,
    callbackGasLimit,
    numWords,
    accId,
    sender,
    l2RequestId,
  ])
  logger.debug(payload, 'payload')

  return {
    payload,
    gasLimit,
    to,
  }
}
