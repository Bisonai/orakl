// 1. Listen on *multiple* smart contracts for a *single* event type.
// 2. Listen on *multiple* smart contracts for *multiple* event types.

import { ethers } from 'ethers'
import * as dotenv from 'dotenv'
import listeners from '../listeners.json' assert { type: 'json' }
dotenv.config()

console.log(listeners)

const provider_url = process.env.PROVIDER
console.log(provider_url)
const provider = new ethers.providers.JsonRpcProvider(provider_url)

const topic = ethers.utils.id('OracleRequest(bytes32,address)')
console.log(topic)

const addresses = listeners.AGGREGATORS

const filterId = await provider.send('eth_newFilter', [
  {
    address: addresses,
    topics: [topic]
  }
])

provider.on('block', async () => {
  const logs = await provider.send('eth_getFilterChanges', [filterId])
  if (logs.length > 1) {
    console.log(logs)
  }
})
