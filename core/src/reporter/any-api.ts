import { Worker } from 'bullmq'
import { ethers } from 'ethers'
import { RequestResponseCoordinator__factory } from '@bisonai-cic/icn-contracts'
import { sendTransaction, buildWallet } from './utils'
import { REPORTER_ANY_API_QUEUE_NAME, BULLMQ_CONNECTION } from '../settings'
import { IAnyApiWorkerReporter } from '../types'

export async function anyApiReporter() {
  console.debug('anyApiReporter')
  const wallet = buildWallet()
  new Worker(REPORTER_ANY_API_QUEUE_NAME, await anyApiJob(wallet), BULLMQ_CONNECTION)
}

function anyApiJob(wallet) {
  const iface = new ethers.utils.Interface(RequestResponseCoordinator__factory.abi)

  async function wrapper(job) {
    const inData: IAnyApiWorkerReporter = job.data
    console.debug('anyApiJob:inData', inData)

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
