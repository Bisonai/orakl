import { VRFCoordinator__factory } from '@bisonai/orakl-contracts/v0.1'
import { processVrfRequest } from '@bisonai/orakl-vrf'
import { Queue, Worker } from 'bullmq'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { getVrfConfig } from '../api'
import {
  BULLMQ_CONNECTION,
  CHAIN,
  NONCE_MANAGER_VRF_QUEUE_NAME,
  VRF_FULFILL_GAS_MINIMUM,
  VRF_FULLFILL_GAS_PER_WORD,
  WORKER_JOB_SETTINGS,
  WORKER_VRF_QUEUE_NAME,
} from '../settings'
import {
  ITransactionParameters,
  IVrfConfig,
  IVrfListenerWorker,
  IVrfTransactionParameters,
  Proof,
  QueueType,
  RequestCommitmentVRF,
} from '../types'
import { remove0x } from '../utils'

const FILE_NAME = import.meta.url

export async function worker(redisClient: RedisClientType, _logger: Logger) {
  const logger = _logger.child({ name: 'worker', file: FILE_NAME })
  const nonceManagerQueue = new Queue(NONCE_MANAGER_VRF_QUEUE_NAME, BULLMQ_CONNECTION)
  // FIXME add checks if exists and if includes all information
  const vrfConfig = await getVrfConfig({ chain: CHAIN, logger })
  const worker = new Worker(
    WORKER_VRF_QUEUE_NAME,
    await job(nonceManagerQueue, vrfConfig, _logger),
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

export async function job(nonceManagerQueue: QueueType, config: IVrfConfig, _logger: Logger) {
  const logger = _logger.child({ name: 'vrfJob', file: FILE_NAME })
  const iface = new ethers.utils.Interface(VRFCoordinator__factory.abi)

  async function wrapper(job) {
    const inData: IVrfListenerWorker = job.data
    logger.debug(inData, 'inData')

    try {
      const alpha = remove0x(
        ethers.utils.solidityKeccak256(['uint256', 'bytes32'], [inData.seed, inData.blockHash]),
      )

      logger.debug({ alpha })
      const { pk, proof, uPoint, vComponents } = processVrfRequest(alpha, config)

      const payloadParameters: IVrfTransactionParameters = {
        blockNum: inData.blockNum,
        seed: inData.seed,
        accId: inData.accId,
        callbackGasLimit: inData.callbackGasLimit,
        numWords: inData.numWords,
        sender: inData.sender,
        isDirectPayment: inData.isDirectPayment,
        pk,
        proof,
        preSeed: inData.seed,
        uPoint,
        vComponents,
      }

      const to = inData.callbackAddress
      const tx = buildTransaction(
        payloadParameters,
        to,
        VRF_FULFILL_GAS_MINIMUM + VRF_FULLFILL_GAS_PER_WORD * inData.numWords,
        iface,
        logger,
      )
      logger.debug(tx, 'tx')

      await nonceManagerQueue.add('vrf', tx, {
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
  payloadParameters: IVrfTransactionParameters,
  to: string,
  gasMinimum: number,
  iface: ethers.utils.Interface,
  logger: Logger,
): ITransactionParameters {
  const gasLimit = payloadParameters.callbackGasLimit + gasMinimum
  const rc: RequestCommitmentVRF = [
    payloadParameters.blockNum,
    payloadParameters.accId,
    payloadParameters.callbackGasLimit,
    payloadParameters.numWords,
    payloadParameters.sender,
  ]
  logger.debug(rc, 'rc')

  const proof: Proof = [
    payloadParameters.pk,
    payloadParameters.proof,
    payloadParameters.preSeed,
    payloadParameters.uPoint,
    payloadParameters.vComponents,
  ]
  logger.debug(proof, 'proof')

  const payload = iface.encodeFunctionData('fulfillRandomWords', [
    proof,
    rc,
    payloadParameters.isDirectPayment,
  ])

  return {
    payload,
    gasLimit,
    to,
  }
}
