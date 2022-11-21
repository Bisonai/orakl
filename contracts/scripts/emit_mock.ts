import { ethers } from 'hardhat'

async function main() {
  const [owner] = await ethers.getSigners()

  const addresses = [
    '0xe78A0F7E598Cc8b0Bb87894B0F60dD2a88d6a8Ab',
    '0x5b1869D9A4C187F2EAa108f3062412ecf0526b24'
  ]

  for (let i = 0; i < addresses.length; ++i) {
    const EventEmitter = await ethers.getContractFactory('EventEmitterMock')
    const eventEmitter = await EventEmitter.attach(addresses[i])

    // parameters
    const specId = ethers.utils.id(12)
    const requester = owner.address

    const nonce = await ethers.provider.getTransactionCount(owner.address)
    const tx = await eventEmitter.buildRequest(specId, requester, { nonce })
    const txReceipt = await tx.wait()
    console.log(txReceipt)
  }
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
