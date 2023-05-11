import { ethers } from 'ethers'
import { Worker, Queue } from 'bullmq'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { VRFCoordinator__factory } from '@bisonai/orakl-contracts'
import { prove, decode, getFastVerifyComponents } from '@bisonai/orakl-vrf'
import {
  RequestCommitmentVRF,
  Proof,
  IVrfResponse,
  IVrfListenerWorker,
  IVrfConfig,
  ITransactionParameters,
  IVrfTransactionParameters,
  QueueType
} from '../types'
import {
  WORKER_VRF_QUEUE_NAME,
  REPORTER_VRF_QUEUE_NAME,
  BULLMQ_CONNECTION,
  CHAIN,
  VRF_FULFILL_GAS_MINIMUM,
  WORKER_JOB_SETTINGS
} from '../settings'
import { getVrfConfig } from '../api'
import { remove0x } from '../utils'

const FILE_NAME = import.meta.url

export async function worker(redisClient: RedisClientType, _logger: Logger) {
  const logger = _logger.child({ name: 'worker', file: FILE_NAME })
  const queue = new Queue(REPORTER_VRF_QUEUE_NAME, BULLMQ_CONNECTION)
  //FIXME add checks if exists and if includes all information
  const vrfConfig = await getVrfConfig({ chain: CHAIN, logger })
  const worker = new Worker(
    WORKER_VRF_QUEUE_NAME,
    await vrfJob(queue, vrfConfig, _logger),
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

export async function vrfJob(queue: QueueType, config: IVrfConfig, _logger: Logger) {
  const logger = _logger.child({ name: 'vrfJob', file: FILE_NAME })

  const iface = new ethers.utils.Interface(VRFCoordinator__factory.abi)

  async function wrapper(job) {
    const inData: IVrfListenerWorker = job.data
    logger.debug(inData, 'inData')

    try {
      const alpha = remove0x(
        ethers.utils.solidityKeccak256(['uint256', 'bytes32'], [inData.seed, inData.blockHash])
      )

      logger.debug({ alpha })
      const { pk, proof, uPoint, vComponents } = processVrfRequest(alpha, config, _logger)

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
        vComponents
      }

      const to = inData.callbackAddress
      const tx = buildTransaction(payloadParameters, to, VRF_FULFILL_GAS_MINIMUM, iface, logger)

      logger.debug(tx, 'tx')

      await queue.add('vrf', tx, {
        jobId: inData.requestId,
        ...WORKER_JOB_SETTINGS
      })

      return tx
    } catch (e) {
      logger.error(e)
    }
  }

  return wrapper
}

function buildTransaction(
  payloadParameters: IVrfTransactionParameters,
  to: string,
  gasMinimum: number,
  iface: ethers.utils.Interface,
  _logger: Logger
): ITransactionParameters {
  const gasLimit = payloadParameters.callbackGasLimit + gasMinimum
  const rc: RequestCommitmentVRF = [
    payloadParameters.blockNum,
    payloadParameters.accId,
    payloadParameters.callbackGasLimit,
    payloadParameters.numWords,
    payloadParameters.sender
  ]
  _logger.debug(rc, 'rc')

  const proof: Proof = [
    payloadParameters.pk,
    payloadParameters.proof,
    payloadParameters.preSeed,
    payloadParameters.uPoint,
    payloadParameters.vComponents
  ]
  _logger.debug(proof, 'proof')

  const payload = iface.encodeFunctionData('fulfillRandomWords', [
    proof,
    rc,
    payloadParameters.isDirectPayment
  ])

  return {
    payload,
    gasLimit,
    to
  }
}

function processVrfRequest(alpha: string, config: IVrfConfig, _logger: Logger): IVrfResponse {
  const logger = _logger.child({ name: 'processVrfRequest', file: FILE_NAME })

  const proof = prove(config.sk, alpha)
  const [Gamma, c, s] = decode(proof)
  const fast = getFastVerifyComponents(config.pk, proof, alpha)

  if (fast == 'INVALID') {
    logger.error('INVALID')
    // TODO throw more specific error
    throw Error()
  }

  return {
    pk: [config.pkX, config.pkY],
    proof: [Gamma.x.toString(), Gamma.y.toString(), c.toString(), s.toString()],
    uPoint: [fast.uX, fast.uY],
    vComponents: [fast.sHX, fast.sHY, fast.cGX, fast.cGY]
  }
}
