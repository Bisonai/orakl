// 1. Listen on *multiple* smart contracts for a *single* event type.
// 2. Listen on *multiple* smart contracts for *multiple* event types.

import { ethers } from 'ethers'
import { Queue } from 'bullmq'
import { ICNOracle__factory, VRFCoordinator__factory } from '@bisonai/icn-contracts'
import { RequestEventData, DataFeedRequest, IListeners, ILog } from './types'
import { IcnError, IcnErrorCode } from './errors'
import { buildBullMqConnection, loadJson } from './utils'
import { WORKER_REQUEST_QUEUE_NAME, WORKER_VRF_QUEUE_NAME } from './settings'
import { PROVIDER_URL, LISTENERS_PATH } from './load-parameters'

async function main() {
  console.log(PROVIDER_URL)
  console.log(ICNOracle__factory.abi)
  console.log(LISTENERS_PATH)

  const listeners = await loadJson(LISTENERS_PATH)
  const provider = new ethers.providers.JsonRpcProvider(PROVIDER_URL)

  const requestIface = new ethers.utils.Interface(ICNOracle__factory.abi)
  const vrfIface = new ethers.utils.Interface(VRFCoordinator__factory.abi)

  listenGetFilterChanges(provider, listeners, requestIface, vrfIface)
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
  requestIface: ethers.utils.Interface,
  vrfIface: ethers.utils.Interface
) {
  const requestQueue = new Queue(WORKER_REQUEST_QUEUE_NAME, buildBullMqConnection())
  const vrfQueue = new Queue(WORKER_VRF_QUEUE_NAME, buildBullMqConnection())

  // Request
  const eventTopicId = getEventTopicId(requestIface.events, 'NewRequest')
  const filterId = await provider.send('eth_newFilter', [
    {
      address: listeners.ANY_API,
      topics: [eventTopicId]
    }
  ])

  // VRF
  const vrfEventTopicId = getEventTopicId(vrfIface.events, 'RandomWordsRequested')
  const vrfFilterId = await provider.send('eth_newFilter', [
    {
      address: listeners.VRF,
      topics: [vrfEventTopicId]
    }
  ])

  console.log(vrfEventTopicId)
  console.log(listeners.VRF)

  provider.on('block', async () => {
    // Request
    const logs: ILog[] = await provider.send('eth_getFilterChanges', [filterId])
    logs.forEach(async (log) => {
      const { requestId, jobId, nonce, callbackAddress, callbackFunctionId, _data } =
        requestIface.parseLog(log).args
      console.log(`requestId ${requestId}`)
      console.log(`jobId ${jobId}`)
      console.log(`nonce ${nonce}`)
      console.log(`callbackAddress ${callbackAddress}`)
      console.log(`callbackFunctionId ${callbackFunctionId}`)
      console.log(`_data ${_data}`)

      // FIXME update name of job
      await requestQueue.add('request', {
        requestId,
        jobId,
        nonce,
        callbackAddress,
        callbackFunctionId,
        _data
      })
    })

    // VRF
    try {
      const vrfLogs: ILog[] = await provider.send('eth_getFilterChanges', [vrfFilterId])
      vrfLogs.forEach(async (log) => {
        try {
          const {
            // keyHash, // FIXME
            requestId,
            preSeed,
            subId,
            minimumRequestConfirmations,
            callbackGasLimit,
            numWords,
            sender
          } = vrfIface.parseLog(log).args
          console.debug('VRF')
          console.debug(log)

          await vrfQueue.add('vrf', {
            callbackAddress: log.address,
            blockNum: log.blockNumber,
            blockHash: log.blockHash,
            requestId,
            seed: preSeed.toString(),
            subId,
            minimumRequestConfirmations,
            callbackGasLimit,
            numWords,
            sender
          })
        } catch (e) {
          console.error(e)
        }

        // TODO add to queue
      })
    } catch (e) {
      console.error(e)
    }
  })
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
