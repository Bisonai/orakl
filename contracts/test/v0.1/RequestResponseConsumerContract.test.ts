import { expect } from 'chai'
import { ethers } from 'hardhat'

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

  it('Request & Fulfill', async function () {
    // Request
    const requestReceipt = await (await userContract.makeRequest()).wait()
    expect(requestReceipt.events.length).to.be.equal(1)
    const requestEvent = requestResponseCoordinator.interface.parseLog(requestReceipt.events[0])
    expect(requestEvent.name).to.be.equal('Requested')

    const eventArgs = [
      'requestId',
      'jobId',
      'nonce',
      'callbackAddress',
      'callbackFunctionId',
      'data'
    ]
    for (const arg of eventArgs) {
      expect(requestEvent.args[arg]).to.not.empty
    }

    // Response
    // TODO change after adding validation node check
    const response = 123
    const { requestId, callbackAddress, callbackFunctionId } = requestEvent.args
    const fulfillReceipt = await (
      await requestResponseCoordinator.fulfillRequestInt256(
        requestId,
        callbackAddress,
        callbackFunctionId,
        response
      )
    ).wait()
    expect(fulfillReceipt.events.length).to.be.equal(1)
    const fulfillEvent = requestResponseCoordinator.interface.parseLog(fulfillReceipt.events[0])
    expect(fulfillEvent.name).to.be.equal('Fulfilled')
    expect(fulfillEvent.args['requestId']).to.be.equal(requestId)
    expect(Number(await userContract.s_response())).to.be.equal(response)
  })
})
