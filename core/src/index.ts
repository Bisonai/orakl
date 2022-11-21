// 1. Listen on *multiple* smart contracts for a *single* event type.
// 2. Listen on *multiple* smart contracts for *multiple* event types.

import { ethers } from 'ethers'
import * as dotenv from 'dotenv'
dotenv.config()

const provider = new ethers.providers.JsonRpcProvider(process.env.PROVIDER)
console.log(provider)

const filter = {
  address: 'dai.tokens.ethers.eth',
  topics: [ethers.utils.id('Transfer(address,address,uint256)')]
}

provider.on(filter, (log, event) => {
  // Emitted whenever a DAI token transfer occurs
})

provider.on('block', (blockNumber) => {
  console.log(blockNumber)
  // Emitted on every block change
})
