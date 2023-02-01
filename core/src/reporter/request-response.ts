import { Worker } from 'bullmq'
import { ethers } from 'ethers'
import { RequestResponseCoordinator__factory } from '@bisonai-cic/icn-contracts'
import { sendTransaction, buildWallet } from './utils'
import { REPORTER_REQUEST_RESPONSE_QUEUE_NAME, BULLMQ_CONNECTION } from '../settings'
import { IRequestResponseWorkerReporter, RequestCommitmentRequestResponse } from '../types'

export async function reporter() {
  console.debug('requestResponse:reporter')
  const wallet = buildWallet()
  new Worker(REPORTER_REQUEST_RESPONSE_QUEUE_NAME, await job(wallet), BULLMQ_CONNECTION)
}

function job(wallet) {
  const iface = new ethers.utils.Interface(RequestResponseCoordinator__factory.abi)

  async function wrapper(job) {
    const inData: IRequestResponseWorkerReporter = job.data
    console.debug('requestResponse:job:inData', inData)

    try {
      const data = typeof inData.data === 'number' ? Math.floor(inData.data) : inData.data
      console.debug('requestResponse:job:data', data)

      const rc: RequestCommitmentRequestResponse = [
        inData.blockNum,
        inData.accId,
        inData.callbackGasLimit,
        inData.sender
      ]

      console.debug('requestResponse:job:requestId', inData.requestId)
      console.debug('requestResponse:job:rc', rc)
      console.debug('requestResponse:job:data', data)
      console.debug('requestResponse:job:isDirectPayment', inData.isDirectPayment)

      const payload = iface.encodeFunctionData('fulfillDataRequest', [
        inData.requestId,
        data,
        rc,
        inData.isDirectPayment
      ])

      await sendTransaction(wallet, inData.callbackAddress, payload)
    } catch (e) {
      console.error(e)
    }
  }

  return wrapper
}
