import { expect } from 'chai'
import pkg from 'hardhat'
const { ethers } = pkg

let requestResponseCoordinator
let userContract

describe('Request-Response cancel request from consumer contract', function () {
  beforeEach(async function () {
    requestResponseCoordinator = await ethers.getContractFactory('RequestResponseCoordinator')
    requestResponseCoordinator = await requestResponseCoordinator.deploy()
    await requestResponseCoordinator.deployed()

    userContract = await ethers.getContractFactory('RequestResponseConsumerMock')
    userContract = await userContract.deploy(requestResponseCoordinator.address)
    await userContract.deployed()
  })

  it('Request & Cancel', async function () {
    // Request
    const requestReceipt = await (await userContract.makeRequest()).wait()
    expect(requestReceipt.events.length).to.be.equal(1)
    const requestEvent = requestResponseCoordinator.interface.parseLog(requestReceipt.events[0])
    expect(requestEvent.name).to.be.equal('Requested')
    const requestId = requestEvent.args['requestId']

    // Cancel
    const cancelReceipt = await (await userContract.cancelRequest(requestId)).wait()
    expect(cancelReceipt.events.length).to.be.equal(1)
    const cancelEvent = requestResponseCoordinator.interface.parseLog(cancelReceipt.events[0])
    expect(cancelEvent.name).to.be.equal('Cancelled')
    expect(cancelEvent.args['requestId']).to.be.equal(requestId)
  })
})
