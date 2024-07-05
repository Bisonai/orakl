import { Aggregator__factory } from '@bisonai/orakl-contracts/v0.1'
import { ethers } from 'ethers'
import { buildMockLogger } from '../src/logger'
import { DATA_FEED_FULFILL_GAS_MINIMUM } from '../src/settings'
import { buildTransaction } from '../src/worker/data-feed.utils'

describe('Data Feed Worker', function () {
  it('Data Feed Build Transaction', async function () {
    const logger = buildMockLogger()
    const oracleAddress = '0xccf9a654c878848991e46ab23d2ad055ca827979' // random address
    const iface = new ethers.utils.Interface(Aggregator__factory.abi)

    const tx = buildTransaction({
      payloadParameters: {
        roundId: 10,
        submission: BigInt(123),
      },
      to: oracleAddress,
      gasMinimum: DATA_FEED_FULFILL_GAS_MINIMUM,
      iface,
      logger,
    })

    expect(tx?.payload).toBe(
      '0x202ee0ed000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000007b',
    )
    expect(tx?.gasLimit).toBe(DATA_FEED_FULFILL_GAS_MINIMUM)
    expect(tx?.to).toBe(oracleAddress)
  })
})
