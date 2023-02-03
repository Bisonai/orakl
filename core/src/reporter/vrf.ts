import { Worker } from 'bullmq'
import { ethers } from 'ethers'
import { VRFCoordinator__factory } from '@bisonai-cic/icn-contracts'
import { sendTransaction, buildWallet } from './utils'
import { REPORTER_VRF_QUEUE_NAME, BULLMQ_CONNECTION } from '../settings'
import { IVrfWorkerReporter, RequestCommitmentVRF, Proof } from '../types'

export async function vrfReporter() {
  console.debug('vrfReporter')
  const wallet = buildWallet()
  new Worker(REPORTER_VRF_QUEUE_NAME, await vrfJob(wallet), BULLMQ_CONNECTION)
}

function vrfJob(wallet) {
  const iface = new ethers.utils.Interface(VRFCoordinator__factory.abi)
  const gasLimit = 3_000_000 // FIXME

  async function wrapper(job) {
    const inData: IVrfWorkerReporter = job.data
    console.debug('vrfJob:inData', inData)

    try {
      const rc: RequestCommitmentVRF = [
        inData.blockNum,
        inData.accId,
        inData.callbackGasLimit,
        inData.numWords,
        inData.sender
      ]
      console.debug('vrfJob:rc', rc)

      const proof: Proof = [
        inData.pk,
        inData.proof,
        inData.preSeed,
        inData.uPoint,
        inData.vComponents
      ]
      console.debug('vrfJob:proof', proof)

      const payload = iface.encodeFunctionData('fulfillRandomWords', [
        proof,
        rc,
        inData.isDirectPayment
      ])
      await sendTransaction(wallet, inData.callbackAddress, payload, gasLimit)
    } catch (e) {
      console.error(e)
    }
  }

  return wrapper
}
