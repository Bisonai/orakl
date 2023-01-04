import { Worker } from 'bullmq'
import { ethers } from 'ethers'
import { Aggregator__factory } from '@bisonai-cic/icn-contracts'
import { sendTransaction, buildWallet } from './utils'
import { REPORTER_AGGREGATOR_QUEUE_NAME, BULLMQ_CONNECTION } from '../settings'
import { IAggregatorWorkerReporter } from '../types'

export async function aggregatorReporter() {
  console.debug('aggregatorReporter')
  const wallet = buildWallet()
  new Worker(REPORTER_AGGREGATOR_QUEUE_NAME, await aggregatorJob(wallet), BULLMQ_CONNECTION)
}

function aggregatorJob(wallet) {
  const iface = new ethers.utils.Interface(Aggregator__factory.abi)

  async function wrapper(job) {
    const inData: IAggregatorWorkerReporter = job.data
    console.debug('aggregatorJob:inData', inData)

    try {
      const payload = iface.encodeFunctionData('submit', [inData.roundId, inData.submission])

      await sendTransaction(wallet, inData.callbackAddress, payload)

      // TODO Put Random Heartbeat job to queue
      // const randomHeartbeat = 3 // seconds
    } catch (e) {
      console.error(e)
    }
  }

  return wrapper
}
