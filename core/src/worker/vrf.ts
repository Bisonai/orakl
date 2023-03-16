import { ethers } from 'ethers'
import { Worker, Queue } from 'bullmq'
import { Logger } from 'pino'
import { prove, decode, getFastVerifyComponents } from '@bisonai/orakl-vrf'
import { IVrfResponse, IVrfListenerWorker, IVrfWorkerReporter, IVrfConfig } from '../types'
import {
  WORKER_VRF_QUEUE_NAME,
  REPORTER_VRF_QUEUE_NAME,
  BULLMQ_CONNECTION,
  CHAIN
} from '../settings'
import { getVrfConfig } from '../api'
import { remove0x } from '../utils'

const FILE_NAME = import.meta.url

export async function vrfWorker(_logger: Logger) {
  _logger.debug({ name: 'vrfWorker', file: FILE_NAME })
  new Worker(
    WORKER_VRF_QUEUE_NAME,
    await vrfJob(REPORTER_VRF_QUEUE_NAME, _logger),
    BULLMQ_CONNECTION
  )
}

async function vrfJob(queueName: string, _logger: Logger) {
  const logger = _logger.child({ name: 'vrfJob', file: FILE_NAME })
  const queue = new Queue(queueName, BULLMQ_CONNECTION)
  // FIXME add checks if exists and if includes all information
  const vrfConfig = await getVrfConfig(CHAIN)

  async function wrapper(job) {
    /*
        const inData: IVrfListenerWorker = job.data
        logger.debug(inData, 'inData')

        try {
      const alpha = remove0x(
        ethers.utils.solidityKeccak256(['uint256', 'bytes32'], [inData.seed, inData.blockHash])
      )

      logger.debug({ alpha })
      const { pk, proof, uPoint, vComponents } = processVrfRequest(alpha, vrfConfig, _logger)

      const outData: IVrfWorkerReporter = {
        callbackAddress: inData.callbackAddress,
        blockNum: inData.blockNum,
        requestId: inData.requestId,
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
      logger.debug(outData, 'outData')

      await queue.add('vrf', outData, {
        jobId: outData.requestId,
        removeOnComplete: {
          age: 1800 // 30 min
        }
      })
    } catch (e) {
      logger.error(e)
    }
      */
  }

  return wrapper
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
    pk: [config.pk_x, config.pk_y],
    proof: [Gamma.x.toString(), Gamma.y.toString(), c.toString(), s.toString()],
    uPoint: [fast.uX, fast.uY],
    vComponents: [fast.sHX, fast.sHY, fast.cGX, fast.cGY]
  }
}
