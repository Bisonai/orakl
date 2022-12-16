// 1. Listen on *multiple* smart contracts for a *single* event type.
// 2. Listen on *multiple* smart contracts for *multiple* event types.

import { ethers } from 'ethers'
import { ICNOracle__factory } from '@bisonai/icn-contracts'
import {
  INewRequest,
  IRandomWordsRequested,
  IAnyApiListenerWorker,
  IVrfListenerWorker,
  IListenerBlock
} from './types'
import { loadJson } from './utils'
import { WORKER_ANY_API_QUEUE_NAME, BULLMQ_CONNECTION } from './settings'
import { PROVIDER_URL, LISTENERS_PATH } from './load-parameters'
import { get_event, get_events } from './get-events'

async function main() {
  console.debug('PROVIDER_URL', PROVIDER_URL)
  console.debug('LISTENERS_PATH', LISTENERS_PATH)
  console.debug('ICNOracle__factory.abi', ICNOracle__factory.abi)

  const listeners = await loadJson(LISTENERS_PATH)
  const provider = new ethers.providers.JsonRpcProvider(PROVIDER_URL)
  const getEvents = new get_event(
    provider,
    listeners.ICN.address,
    WORKER_ANY_API_QUEUE_NAME,
    'NewRequest',
    ICNOracle__factory.abi,
    listeners.ICN.blockFilePath,
    processICNEvent
  )

  let running = false
  setInterval(async () => {
    if (!running) {
      running = true
      await getEvents.get_events()
      running = false
    } else {
      console.log('running')
    }
  }, 500)
}

function processICNEvent(iface, queue) {
  async function wrapper(log) {
    const eventData: INewRequest = iface.parseLog(log).args
    console.debug('processICNEvent:eventData', eventData)
    const data: IAnyApiListenerWorker = {
      oracleCallbackAddress: log.address,
      requestId: eventData.requestId.toString(),
      jobId: eventData.jobId.toString(),
      nonce: eventData.nonce.toString(),
      callbackAddress: eventData.callbackAddress.toString(),
      callbackFunctionId: eventData.callbackFunctionId.toString(),
      _data: eventData._data.toString()
    }
    console.debug('processICNEvent:data', data)

    await queue.add('icn', data)
  }
  return wrapper
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
