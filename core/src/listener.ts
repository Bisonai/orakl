// 1. Listen on *multiple* smart contracts for a *single* event type.
// 2. Listen on *multiple* smart contracts for *multiple* event types.

import { ethers } from 'ethers'
import { Queue } from 'bullmq'
import { ICNOracle__factory, VRFCoordinator__factory } from '../../contracts/typechain-types'
import {
  RequestEventData,
  DataFeedRequest,
  IListeners,
  ILog,
  INewRequest,
  IRandomWordsRequested,
  IAnyApiListenerWorker,
  IVrfListenerWorker
} from './types'
import { IcnError, IcnErrorCode } from './errors'
import { loadJson, readTextFile, writeTextFile } from './utils'
import { WORKER_ANY_API_QUEUE_NAME, WORKER_VRF_QUEUE_NAME, BULLMQ_CONNECTION } from './settings'
import { PROVIDER_URL, LISTENERS_PATH } from './load-parameters'
import { filter } from 'mathjs'
async function main() {
  console.debug('PROVIDER_URL', PROVIDER_URL)
  console.debug('LISTENERS_PATH', LISTENERS_PATH)
  console.debug('ICNOracle__factory.abi', ICNOracle__factory.abi)
  console.debug('VRFCoordinator__factory.abi', VRFCoordinator__factory.abi)

  const listeners = await loadJson(LISTENERS_PATH)
  const provider = new ethers.providers.JsonRpcProvider(PROVIDER_URL)

  const anyApiIface = new ethers.utils.Interface(ICNOracle__factory.abi)
  listenToEvents(
    provider,
    listeners.ANY_API,
    WORKER_ANY_API_QUEUE_NAME,
    'NewRequest',
    anyApiIface,
    processAnyApiEvent
  )

  const vrfIface = new ethers.utils.Interface(VRFCoordinator__factory.abi)
  // listenToEvents(
  //   provider,
  //   listeners.VRF,
  //   WORKER_VRF_QUEUE_NAME,
  //   'RandomWordsRequested',
  //   vrfIface,
  //   processVrfEvent
  // )
  // TODO listen to events for Predefined Feeds
}

function processAnyApiEvent(iface, queue) {
  async function wrapper(log) {
    const eventData: INewRequest = iface.parseLog(log).args
    console.debug('processAnyApiEvent:eventData', eventData)

    const data: IAnyApiListenerWorker = {
      oracleCallbackAddress: log.address,
      requestId: eventData.requestId.toString(),
      jobId: eventData.jobId,
      nonce: eventData.nonce.toString(),
      callbackAddress: eventData.callbackAddress,
      callbackFunctionId: eventData.callbackFunctionId,
      _data: eventData._data // FIXME rename?
    }
    console.debug('processAnyApiEvent:data', data)

    await queue.add('any-api', data)
  }

  return wrapper
}

// TODO
function processPredefinedFeedEvent(iface, queue) {
  async function wrapper(log) {
    const eventData = iface.parseLog(log).args
    console.debug('processPredefinedEvent:eventData', eventData)

    const data /*: IPredefinedFeedListenerWorker */ = {}
    console.debug('processPredefinedEvent:data', data)

    await queue.add('predefined-feed', data)
  }
  return wrapper
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

function getEventTopicId(
  events: { [name: string]: ethers.utils.EventFragment },
  eventName: string
): string {
  console.debug('getEventTopicId:events', events)
  for (const [key, value] of Object.entries(events)) {
    if (value.name == eventName) {
      return ethers.utils.id(key)
    }
  }

  throw new IcnError(IcnErrorCode.NonExistantEventError, `Event [${eventName}] not found.`)
}

async function listenToEvents(
  provider: ethers.providers.JsonRpcProvider,
  listeners: IListeners,
  queueName: string,
  eventName: string,
  iface: ethers.utils.Interface,
  wrapFn
) {
  const queue = new Queue(queueName, BULLMQ_CONNECTION)
  const topicId = getEventTopicId(iface.events, eventName)
  const fn = wrapFn(iface, queue)
  const filterId = await provider.send('eth_newFilter', [
    {
      address: listeners,
      topics: [topicId]
    }
  ])

  console.debug(`listenToEvents:topicId ${topicId}`)
  console.debug(`listenToEvents:listeners ${listeners}`)

  // provider.on('block', async (blockNumber) => {
  //   console.log('listen block',blockNumber)
  //   // const logs: ILog[] = await provider.send('eth_getFilterChanges', [filterId])
  //   // logs.forEach(fn)
  // })
  const emit_contract = new ethers.Contract('0x45778c29A34bA00427620b937733490363839d8C', ICNOracle__factory.abi, provider);

  //   emit_contract.on(eventName,(_requestId, _jobId, _nonce, _callbackAddress, _callbackFunctionId, _data) =>{
  //     console.log({_requestId, _jobId, _nonce, _callbackAddress, _callbackFunctionId, _data});
  // });
  let running = false;
  setInterval(async () => {
    if (!running) {
      running = true

      const logs = await get_events(eventName, emit_contract, provider)
      logs.forEach(fn)
      console.log('logs', logs);
      running = false;
    }
    else {
      console.log('running');
    }
  }, 500);
}

let start_block: number = 0;
let end_block: number = 0;
const block_file_path = 'src/data/block.txt';

async function get_events(eventName: string, emit_contract, provider) {
  try {
    //const emit_contract = new ethers.Contract('0x45778c29A34bA00427620b937733490363839d8C', ICNOracle__factory.abi, provider);

    if (start_block <= 0) {
      const ct = await readTextFile(block_file_path);
      start_block = parseInt(ct);
    }
    const latest_block = await provider.getBlockNumber();
    if (end_block < latest_block)
      end_block = latest_block;
    if (latest_block > (start_block + 1)) {
      console.log(start_block + 1, ' - ', latest_block);

      const events = await emit_contract.queryFilter(eventName, start_block + 1, latest_block);
      //console.log(events);
      //save last block here
      await writeTextFile(block_file_path, start_block.toString());
      start_block = latest_block;
      return events;
    }
    else console.log('already get data');
  } catch (error) {
    console.log(error);
  }
  return [];
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
