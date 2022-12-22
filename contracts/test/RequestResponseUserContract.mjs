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

    userContract = await ethers.getContractFactory('RequestResponseMock')
    userContract = await userContract.deploy(requestResponseCoordinator.address)
    await userContract.deployed()
  })

  it('Should be able to request data', async function () {
    await expect(userContract.requestData()).to.not.be.reverted
  })

  it('Should emit event NewRequest', async function () {
    const txReceipt = await (await userContract.requestData()).wait()

    expect(txReceipt.events.length).to.be.equal(1)

    const event = requestResponseCoordinator.interface.parseLog(txReceipt.events[0])
    expect(event.name).to.be.equal('NewRequest')

    const eventArgs = [
      'requestId',
      'jobId',
      'nonce',
      'callbackAddress',
      'callbackFunctionId',
      '_data'
    ]

    for (const arg of eventArgs) {
      expect(event.args[arg]).to.not.empty
    }
  })
})
