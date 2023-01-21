import { ethers } from 'ethers'
import { Worker, Queue } from 'bullmq'
import { prove, decode, getFastVerifyComponents } from '../vrf/index'
import { IVrfResponse, IVrfListenerWorker, IVrfWorkerReporter, IVrfConfig } from '../types'
import {
  WORKER_VRF_QUEUE_NAME,
  REPORTER_VRF_QUEUE_NAME,
  BULLMQ_CONNECTION,
  DB,
  CHAIN,
  getVrfConfig
} from '../settings'
import { remove0x } from '../utils'

export async function vrfWorker() {
  console.debug('vrfWorker')
  new Worker(WORKER_VRF_QUEUE_NAME, await vrfJob(REPORTER_VRF_QUEUE_NAME), BULLMQ_CONNECTION)
}

async function vrfJob(queueName) {
  const queue = new Queue(queueName, BULLMQ_CONNECTION)
  // FIXME add checks if exists and if includes all information
  const vrfConfig = await getVrfConfig(DB, CHAIN)

  async function wrapper(job) {
    const inData: IVrfListenerWorker = job.data
    console.debug('vrfJob:inData', inData)

    try {
      const alpha = remove0x(
        ethers.utils.solidityKeccak256(['uint256', 'bytes32'], [inData.seed, inData.blockHash])
      )

      console.debug('vrfJob:alpha', alpha)
      const { pk, proof, uPoint, vComponents } = processVrfRequest(alpha, vrfConfig)

      const outData: IVrfWorkerReporter = {
        callbackAddress: inData.callbackAddress,
        blockNum: inData.blockNum,
        requestId: inData.requestId,
        seed: inData.seed,
        accId: inData.accId,
        minimumRequestConfirmations: inData.minimumRequestConfirmations,
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
      console.debug('vrfJob:outData', outData)

      await queue.add('vrf', outData)
    } catch (e) {
      console.error(e)
    }
  }

  return wrapper
}

function processVrfRequest(alpha: string, config: IVrfConfig): IVrfResponse {
  console.debug('processVrfRequest:alpha', alpha)

  const proof = prove(config.sk, alpha)
  const [Gamma, c, s] = decode(proof)
  const fast = getFastVerifyComponents(config.pk, proof, alpha)

  if (fast == 'INVALID') {
    console.error('INVALID')
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
