import pkg from 'hardhat'
import assert from 'node:assert'
const { ethers } = pkg

async function main() {
  const [owner] = await ethers.getSigners()

  const count = 2
  const addresses = [
    '0xe78A0F7E598Cc8b0Bb87894B0F60dD2a88d6a8Ab',
    '0x5b1869D9A4C187F2EAa108f3062412ecf0526b24'
  ]
  const specIds = [...Array(count)].map((i) => ethers.utils.id(String(i)))
  const requesters = [...Array(count)].map(() => ethers.Wallet.createRandom().address)
  const payments = [...Array(count)].map(() => Math.floor(Math.random() * 10))

  assert(addresses.length == count)
  assert(specIds.length == count)
  assert(requesters.length == count)
  assert(payments.length == count)

  for (let i = 0; i < count; ++i) {
    const EventEmitter = await ethers.getContractFactory('EventEmitterMock')
    const eventEmitter = await EventEmitter.attach(addresses[i])

    const nonce = await ethers.provider.getTransactionCount(owner.address)
    const tx = await eventEmitter.buildRequest(specIds[i], requesters[i], payments[i], { nonce })
    const txReceipt = await tx.wait()
    console.log(txReceipt)
  }
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
