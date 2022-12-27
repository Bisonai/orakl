import { expect } from 'chai'
import pkg from 'hardhat'
const { ethers } = pkg

let requestResponseCoordinator
let userContract

describe('Request-Response user contract', function () {
  beforeEach(async function () {
    requestResponseCoordinator = await ethers.getContractFactory('RequestResponseCoordinator')
    requestResponseCoordinator = await requestResponseCoordinator.deploy()
    await requestResponseCoordinator.deployed()

    userContract = await ethers.getContractFactory('RequestResponseConsumerMock')
    userContract = await userContract.deploy(requestResponseCoordinator.address)
    await userContract.deployed()
  })

  it('Should be able to request data', async function () {
    await expect(userContract.makeRequest()).to.not.be.reverted
  })

  it('Should emit event NewRequest', async function () {
    const txReceipt = await (await userContract.makeRequest()).wait()

    expect(txReceipt.events.length).to.be.equal(1)

    const event = requestResponseCoordinator.interface.parseLog(txReceipt.events[0])
    expect(event.name).to.be.equal('Requested')

    const eventArgs = [
      'requestId',
      'jobId',
      'nonce',
      'callbackAddress',
      'callbackFunctionId',
      'data'
    ]

    for (const arg of eventArgs) {
      expect(event.args[arg]).to.not.empty
    }
  })
})
