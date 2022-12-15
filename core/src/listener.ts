// 1. Listen on *multiple* smart contracts for a *single* event type.
// 2. Listen on *multiple* smart contracts for *multiple* event types.

import path from 'node:path'
import { Contract, Event, ContractInterface, ethers } from 'ethers'
import { Queue } from 'bullmq'
import { ICNOracle__factory, VRFCoordinator__factory } from '@bisonai/icn-contracts'
import {
  RequestEventData,
  DataFeedRequest,
  IListeners,
  ILog,
  INewRequest,
  IRandomWordsRequested,
  IAnyApiListenerWorker,
  IVrfListenerWorker,
  IListenerBlock
} from './types'
import { IcnError, IcnErrorCode } from './errors'
import { loadJson, readTextFile, writeTextFile } from './utils'
import {
  WORKER_ANY_API_QUEUE_NAME,
  WORKER_VRF_QUEUE_NAME,
  BULLMQ_CONNECTION,
  LISTENER_ROOT_DIR,
  LISTENER_DELAY
} from './settings'
import { PROVIDER_URL, LISTENERS_PATH } from './load-parameters'

async function main() {
  console.debug('PROVIDER_URL', PROVIDER_URL)
  console.debug('LISTENERS_PATH', LISTENERS_PATH)
  console.debug('ICNOracle__factory.abi', ICNOracle__factory.abi)
  console.debug('VRFCoordinator__factory.abi', VRFCoordinator__factory.abi)

  const listeners = await loadJson(LISTENERS_PATH)
  const provider = new ethers.providers.JsonRpcProvider(PROVIDER_URL)

  listenToEvents(
    provider,
    listeners.ANY_API,
    WORKER_ANY_API_QUEUE_NAME,
    'NewRequest',
    ICNOracle__factory.abi,
    processAnyApiEvent
  )

  // TODO listen to events for Predefined Feeds

  listenToEvents(
    provider,
    listeners.VRF,
    WORKER_VRF_QUEUE_NAME,
    'RandomWordsRequested',
    VRFCoordinator__factory.abi,
    processVrfEvent
  )
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
  listeners: Array<string>,
  queueName: string,
  eventName: string,
  abi: Array<object>,
  wrapFn
) {
  const iface = new ethers.utils.Interface(abi)
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

  const listener = listeners[0]
  const emitContract = new ethers.Contract(listeners[0], abi, provider)

  const lstener_block: IListenerBlock = {
    startBlock: 0,
    filePath: path.join(LISTENER_ROOT_DIR, `${listener}.txt`)
  }

  let running = false
  setInterval(async () => {
    if (!running) {
      running = true
      const logs = await get_events(eventName, emitContract, provider, lstener_block)
      logs.forEach(fn)
      if (logs.length > 0) console.debug('logs', logs)
      running = false
    }
  }, LISTENER_DELAY)
}

async function get_events(
  eventName: string,
  emitContract: Contract,
  provider: ethers.providers.JsonRpcProvider,
  listenerBlock: IListenerBlock
) {
  try {
    let events: Event[] = []
    if (listenerBlock.startBlock <= 0) {
      let start = 0
      try {
        start = parseInt(await readTextFile(listenerBlock.filePath))
      } catch (e) {
        console.error(e)
      }

      listenerBlock.startBlock = start
    }

    const latest_block = await provider.getBlockNumber()
    console.debug(`${listenerBlock.startBlock} - ${latest_block}`)

    if (latest_block >= listenerBlock.startBlock) {
      events = await emitContract.queryFilter(eventName, listenerBlock.startBlock, latest_block)
      console.debug(`${listenerBlock.startBlock} - ${latest_block}`)

      listenerBlock.startBlock = latest_block + 1
      await writeTextFile(listenerBlock.filePath, listenerBlock.startBlock.toString())
    }

    return events
  } catch (e) {
    console.log(e)
    throw Error(e)
  }
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
