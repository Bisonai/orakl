// 1. Listen on *multiple* smart contracts for a *single* event type.
// 2. Listen on *multiple* smart contracts for *multiple* event types.

import { ethers } from 'ethers'
import * as dotenv from 'dotenv'
import listeners from '../listeners.json' assert { type: 'json' }
// FIXME load from @bisonai/icn-contracts
import EventEmitterMockJson from '../../contracts/artifacts/src/v0.1/mocks/EventEmitterMock.sol/EventEmitterMock.json' assert { type: 'json' }
import { RequestEventData } from './types'

dotenv.config()

async function main() {
  const provider_url = process.env.PROVIDER

  console.log(provider_url)
  console.log(EventEmitterMockJson.abi)

  const provider = new ethers.providers.JsonRpcProvider(provider_url)
  const iface = new ethers.utils.Interface(EventEmitterMockJson.abi)

  listenGetFilterChanges(provider, listeners, iface)
}

function getEventTopicId(events, eventName: string): string {
  for (const [key, value] of Object.entries(events)) {
    if (value.name == eventName) {
      return ethers.utils.id(key)
    }
  }
}

async function listenGetFilterChanges(provider, listeners, iface) {
  const eventTopicId = getEventTopicId(iface.events, 'OracleRequest')

  const filterId = await provider.send('eth_newFilter', [
    {
      address: listeners.AGGREGATORS,
      topics: [eventTopicId]
    }
  ])

  provider.on('block', async () => {
    const logs = await provider.send('eth_getFilterChanges', [filterId])

    logs.forEach((log) => {
      const { specId, requester, payment }: RequestEventData = iface.parseLog(log).args
      console.log(`specId ${specId}`)
      console.log(`requester ${requester}`)
      console.log(`payment ${payment.toString()}`)
    })
  })
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
