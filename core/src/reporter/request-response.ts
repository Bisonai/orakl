import { Worker } from 'bullmq'
import { ethers } from 'ethers'
import { RequestResponseCoordinator__factory } from '@bisonai-cic/icn-contracts'
import { sendTransaction, buildWallet } from './utils'
import { REPORTER_REQUEST_RESPONSE_QUEUE_NAME, BULLMQ_CONNECTION } from '../settings'
import { IRequestResponseWorkerReporter } from '../types'

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
      const _data = typeof inData.data === 'number' ? Math.floor(inData.data) : inData.data

      const payload = iface.encodeFunctionData('fulfillOracleRequest', [
        inData.requestId,
        inData.callbackAddress,
        inData.callbackFunctionId,
        _data
      ])

      await sendTransaction(wallet, inData.oracleCallbackAddress, payload)
    } catch (e) {
      console.error(e)
    }
  }

  return wrapper
}
