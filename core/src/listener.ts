// 1. Listen on *multiple* smart contracts for a *single* event type.
// 2. Listen on *multiple* smart contracts for *multiple* event types.

import { ethers } from 'ethers'
import { Queue } from 'bullmq'
import { ICNOracle__factory, VRFCoordinator__factory } from '@bisonai/icn-contracts'
import {
  RequestEventData,
  DataFeedRequest,
  IListeners,
  ILog,
  INewRequest,
  IRandomWordsRequested
} from './types'
import { IcnError, IcnErrorCode } from './errors'
import { loadJson } from './utils'
import { WORKER_REQUEST_QUEUE_NAME, WORKER_VRF_QUEUE_NAME, BULLMQ_CONNECTION } from './settings'
import { PROVIDER_URL, LISTENERS_PATH } from './load-parameters'

async function main() {
  console.debug('PROVIDER_URL', PROVIDER_URL)
  console.debug('LISTENERS_PATH', LISTENERS_PATH)
  console.debug('ICNOracle__factory.abi', ICNOracle__factory.abi)
  console.debug('VRFCoordinator__factory.abi', VRFCoordinator__factory.abi)

  const listeners = await loadJson(LISTENERS_PATH)
  const provider = new ethers.providers.JsonRpcProvider(PROVIDER_URL)

  const anyApiIface = new ethers.utils.Interface(ICNOracle__factory.abi)
  listenGetFilterChanges(
    provider,
    listeners.ANY_API,
    WORKER_REQUEST_QUEUE_NAME,
    'NewRequest',
    anyApiIface,
    processAnyApiEvent
  )

  const vrfIface = new ethers.utils.Interface(VRFCoordinator__factory.abi)
  listenGetFilterChanges(
    provider,
    listeners.VRF,
    WORKER_VRF_QUEUE_NAME,
    'RandomWordsRequested',
    vrfIface,
    processVrfEvent
  )
}

function processAnyApiEvent(iface, queue) {
  async function wrapper(log) {
    const eventData: INewRequest = iface.parseLog(log).args
    console.debug('NewRequest', eventData)

    await queue.add('anyApi', {
      requestId: eventData.requestId,
      jobId: eventData.jobId,
      nonce: eventData.nonce,
      callbackAddress: eventData.callbackAddress,
      callbackFunctionId: eventData.callbackFunctionId,
      _data: eventData._data // FIXME rename?
    })
  }

  return wrapper
}

function processVrfEvent(iface, queue) {
  async function wrapper(log) {
    const eventData: IRandomWordsRequested = iface.parseLog(log).args
    console.debug('RequestRandomWords', eventData)

    await queue.add('vrf', {
      callbackAddress: log.address,
      blockNum: log.blockNumber,
      blockHash: log.blockHash,
      requestId: eventData.requestId,
      seed: eventData.preSeed.toString(),
      subId: eventData.subId,
      minimumRequestConfirmations: eventData.minimumRequestConfirmations,
      callbackGasLimit: eventData.callbackGasLimit,
      numWords: eventData.numWords,
      sender: eventData.sender
    })
  }

  return wrapper
}

function getEventTopicId(
  events: { [name: string]: ethers.utils.EventFragment },
  eventName: string
): string {
  console.log(events)
  for (const [key, value] of Object.entries(events)) {
    if (value.name == eventName) {
      return ethers.utils.id(key)
    }
  }

  throw new IcnError(IcnErrorCode.NonExistantEventError, `Event [${eventName}] not found.`)
}

async function listenGetFilterChanges(
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

  console.debug(`listenGetFilterChanges:topicId ${topicId}`)
  console.debug(`listenGetFilterChanges:listeners ${listeners}`)

  provider.on('block', async () => {
    const logs: ILog[] = await provider.send('eth_getFilterChanges', [filterId])
    logs.forEach(fn)
  })
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
