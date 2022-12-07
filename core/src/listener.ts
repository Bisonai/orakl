// 1. Listen on *multiple* smart contracts for a *single* event type.
// 2. Listen on *multiple* smart contracts for *multiple* event types.

import { ethers } from 'ethers'
import { Queue } from 'bullmq'
import { ICNOracle__factory } from '@bisonai/icn-contracts'
import { RequestEventData, DataFeedRequest, IListeners, ILog } from './types'
import { IcnError, IcnErrorCode } from './errors'
import { buildBullMqConnection, loadJson } from './utils'
import { workerRequestQueueName } from './settings'
import { PROVIDER_URL, LISTENERS_PATH } from './load-parameters'

async function main() {
  console.log(PROVIDER_URL)
  console.log(ICNOracle__factory.abi)
  console.log(LISTENERS_PATH)

  const listeners = await loadJson(LISTENERS_PATH)
  const provider = new ethers.providers.JsonRpcProvider(PROVIDER_URL)
  const iface = new ethers.utils.Interface(ICNOracle__factory.abi)

  listenGetFilterChanges(provider, listeners, iface)
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
  iface: ethers.utils.Interface
) {
  const eventTopicId = getEventTopicId(iface.events, 'NewRequest')
  const filterId = await provider.send('eth_newFilter', [
    {
      address: listeners.ANY_API,
      topics: [eventTopicId]
    }
  ])

  const queue = new Queue(workerRequestQueueName, buildBullMqConnection())

  provider.on('block', async () => {
    const logs: ILog[] = await provider.send('eth_getFilterChanges', [filterId])
    logs.forEach(async (log) => {
      const { requestId, jobId, nonce, callbackAddress, callbackFunctionId, _data } =
        iface.parseLog(log).args
      console.log(`requestId ${requestId}`)
      console.log(`jobId ${jobId}`)
      console.log(`nonce ${nonce}`)
      console.log(`callbackAddress ${callbackAddress}`)
      console.log(`callbackFunctionId ${callbackFunctionId}`)
      console.log(`_data ${_data}`)

      // FIXME update name of job
      await queue.add(jobId, {
        requestId,
        jobId,
        nonce,
        callbackAddress,
        callbackFunctionId,
        _data
      })
      // await queue.add('myJobName', { specId, requester, payment })
    })
  })
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
