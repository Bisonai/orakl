import { ethers } from 'hardhat'

async function main() {
  const [owner] = await ethers.getSigners()

  const EventEmitter = await ethers.getContractFactory('EventEmitterMock')
  const numContracts = 2
  for (let i = 0; i < numContracts; ++i) {
    const eventEmitter = await EventEmitter.deploy()

    await eventEmitter.deployed()
    console.log('EventEmitterMock deployed to:', eventEmitter.address)
  }
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
