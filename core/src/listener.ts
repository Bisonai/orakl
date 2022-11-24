// 1. Listen on *multiple* smart contracts for a *single* event type.
// 2. Listen on *multiple* smart contracts for *multiple* event types.

import { ethers } from 'ethers'
import * as dotenv from 'dotenv'
import { Queue } from 'bullmq'
import { EventEmitterMock__factory } from '@bisonai/icn-contracts'
import { RequestEventData, DataFeedRequest, IListeners, ILog } from './types.js'
import { IcnError, IcnErrorCode } from './errors.js'
import { buildBullMqConnection, buildQueueName, loadJson } from './utils.js'

dotenv.config()

async function main() {
  const provider_url = process.env.PROVIDER
  const listeners_path = process.env.LISTENERS // FIXME raise error when file does not exist

  console.log(provider_url)
  console.log(EventEmitterMock__factory.abi)
  console.log(listeners_path)

  const listeners = await loadJson(listeners_path)
  const provider = new ethers.providers.JsonRpcProvider(provider_url)
  const iface = new ethers.utils.Interface(EventEmitterMock__factory.abi)

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
  const eventTopicId = getEventTopicId(iface.events, 'OracleRequest')
  const filterId = await provider.send('eth_newFilter', [
    {
      address: listeners.AGGREGATORS,
      topics: [eventTopicId]
    }
  ])

  const queue = new Queue(buildQueueName(), buildBullMqConnection())

  provider.on('block', async () => {
    const logs: ILog[] = await provider.send('eth_getFilterChanges', [filterId])

    logs.forEach(async (log) => {
      const { specId, requester, payment } = iface.parseLog(log).args
      console.log(`specId ${specId}`)
      console.log(`requester ${requester}`)
      console.log(`payment ${payment.toString()}`)

      await queue.add('myJobName', { specId, requester, payment })
    })
  })
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
