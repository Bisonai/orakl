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
  IRandomWordsRequested,
  IAnyApiListenerWorker,
  IVrfListenerWorker
} from './types'
import { IcnError, IcnErrorCode } from './errors'
import { loadJson } from './utils'
import {
  WORKER_ANY_API_QUEUE_NAME,
  WORKER_VRF_QUEUE_NAME,
  WORKER_AGGREGATOR_QUEUE_NAME,
  BULLMQ_CONNECTION
} from './settings'
import { PROVIDER_URL, LISTENERS_PATH } from './load-parameters'

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

  // TODO listen to events for Predefined Feeds

  const vrfIface = new ethers.utils.Interface(VRFCoordinator__factory.abi)
  listenToEvents(
    provider,
    listeners.VRF,
    WORKER_VRF_QUEUE_NAME,
    'RandomWordsRequested',
    vrfIface,
    processVrfEvent
  )

  // TODO load from contract ABI
  const ICNOracleAggreagatorAbi = [
    {
      anonymous: false,
      inputs: [
        {
          indexed: false,
          internalType: 'uint256',
          name: 'answerCounter',
          type: 'uint256'
        },
        {
          indexed: false,
          internalType: 'address',
          name: 'aggregatorAddress',
          type: 'address'
        },
        {
          indexed: false,
          internalType: 'uint256',
          name: 'roundTimestamp',
          type: 'uint256'
        }
      ],
      name: 'NewRound',
      type: 'event'
    }
  ]
  const aggreagatorIface = new ethers.utils.Interface(ICNOracleAggreagatorAbi)
  listenToEvents(
    provider,
    // FIXME get aggregator list from ./aggregator/*.aggregator.json
    listeners.AGGREAGATOR,
    WORKER_AGGREGATOR_QUEUE_NAME,
    'NewRound',
    aggreagatorIface,
    processAggregatorEvent
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

function processAggregatorEvent(iface, queue) {
  async function wrapper(log) {
    const eventData /* : INewRound */ = iface.parseLog(log).args
    console.debug('processAggregatorEvent:eventData', eventData)

    // TODO Remove relevant delayed jobs from Fixed Heartbeast queue.
    // TODO Remove relevant delayed jobs from Random Heartbeat queue.
    // TODO Did I cause this emitted event? If yes put back Random Heartbeat and exit.

    const data /* : IAggregatorListenerWorker */ = {
      mustReport: true,
      answerCounter: eventData.answerCounter, // FIXME Why do we need it?
      aggregatorAddress: eventData.answerCounter, // FIXME Why do we need it?
      roundTimestamp: eventData.roundTimestamp // FIXME why not roundId
    }
    console.debug('processAggregatorEvent:data', data)

    await queue.add('aggregator', data)
    // TODO Add Fixed Heartbeast job to relevant queue.
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

  provider.on('block', async () => {
    const logs: ILog[] = await provider.send('eth_getFilterChanges', [filterId])
    logs.forEach(fn)
  })
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
