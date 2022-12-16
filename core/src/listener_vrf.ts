// 1. Listen on *multiple* smart contracts for a *single* event type.
// 2. Listen on *multiple* smart contracts for *multiple* event types.

import { ethers } from 'ethers'
import { Queue } from 'bullmq'
import { ICNOracle__factory, VRFCoordinator__factory } from '../../contracts/typechain-types'
import {
  INewRequest,
  IRandomWordsRequested,
  IAnyApiListenerWorker,
  IVrfListenerWorker,
  IListenerBlock
} from './types'
import { loadJson} from './utils'
import { WORKER_ANY_API_QUEUE_NAME, WORKER_VRF_QUEUE_NAME, BULLMQ_CONNECTION } from './settings'
import { PROVIDER_URL, LISTENERS_PATH } from './load-parameters'
import { get_event, get_events } from './get-events'

async function main() {
  console.debug('PROVIDER_URL', PROVIDER_URL)
  console.debug('LISTENERS_PATH', LISTENERS_PATH)
  console.debug('ICNOracle__factory.abi', ICNOracle__factory.abi)
  console.debug('VRFCoordinator__factory.abi', VRFCoordinator__factory.abi)

  const listeners = await loadJson(LISTENERS_PATH)
  const provider = new ethers.providers.JsonRpcProvider(PROVIDER_URL)
  const getEvents=new get_event(provider,listeners.VRF.address,WORKER_VRF_QUEUE_NAME,
    'RandomWordsRequested',
    VRFCoordinator__factory.abi,
    listeners.VRF.blockFilePath,
    processVrfEvent)

  let running=false;
  setInterval(async () => {
    if (!running) {
      //console.log('Ready running');
      running = true
      await getEvents.get_events();
      running = false;
    }
    else {
      console.log('running');
    }
  }, 500);
}


function processVrfEvent(iface, queue) {
  async function wrapper(log) {
    const eventData: IRandomWordsRequested = iface.parseLog(log).args
    console.debug('processVrfEvent:eventData', eventData)
    const data: IVrfListenerWorker = {
      callbackAddress: log.address,
      blockNum: log.blockNumber,
      blockHash: log.blockHash,
      requestId: eventData.requestId.toString(),
      seed: eventData.preSeed.toString(),
      subId: eventData.subId.toString(),
      minimumRequestConfirmations: eventData.minimumRequestConfirmations,
      callbackGasLimit: eventData.callbackGasLimit,
      numWords: eventData.numWords,
      sender: eventData.sender
    }
    console.debug('processVrfEvent:data', data)

    await queue.add('vrf', data)
  }
  return wrapper
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})


