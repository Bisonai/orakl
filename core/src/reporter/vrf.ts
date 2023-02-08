import { Worker } from 'bullmq'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import { VRFCoordinator__factory } from '@bisonai/orakl-contracts'
import { loadWalletParameters, sendTransaction, buildWallet } from './utils'
import { REPORTER_VRF_QUEUE_NAME, BULLMQ_CONNECTION } from '../settings'
import { IVrfWorkerReporter, RequestCommitmentVRF, Proof } from '../types'

const FILE_NAME = import.meta.url

export async function reporter(_logger: Logger) {
  _logger.debug({ name: 'vrfrReporter', file: FILE_NAME })
  const { privateKey, providerUrl } = loadWalletParameters()
  const wallet = await buildWallet({ privateKey, providerUrl })
  new Worker(REPORTER_VRF_QUEUE_NAME, await job(wallet, _logger), BULLMQ_CONNECTION)
}

function job(wallet, _logger: Logger) {
  const logger = _logger.child({ name: 'job', file: FILE_NAME })
  const iface = new ethers.utils.Interface(VRFCoordinator__factory.abi)
  const gasLimit = 3_000_000 // FIXME

  async function wrapper(job) {
    const inData: IVrfWorkerReporter = job.data
    logger.debug(inData, 'inData')

    try {
      const rc: RequestCommitmentVRF = [
        inData.blockNum,
        inData.accId,
        inData.callbackGasLimit,
        inData.numWords,
        inData.sender
      ]
      logger.debug(rc, 'rc')

      const proof: Proof = [
        inData.pk,
        inData.proof,
        inData.preSeed,
        inData.uPoint,
        inData.vComponents
      ]
      logger.debug(proof, 'proof')

      const payload = iface.encodeFunctionData('fulfillRandomWords', [
        proof,
        rc,
        inData.isDirectPayment
      ])
      await sendTransaction({ wallet, to: inData.callbackAddress, payload, gasLimit, _logger })
    } catch (e) {
      logger.error(e)
    }
  }

  return wrapper
}
