const { expect } = require('chai')
const { ethers } = require('hardhat')
const { loadFixture } = require('@nomicfoundation/hardhat-network-helpers')

const { createSigners } = require('../utils.cjs')
const { requestResponseConfig } = require('./RequestResponse.config.cjs')

async function deploy() {
  const {
    account0: deployerSigner,
    account1: consumerSigner,
    account2,
    account3,
    account4,
    account5,
  } = await createSigners()

  // L2 endpoint
  let l2EndpointContract = await ethers.getContractFactory('L2Endpoint', { signer: deployerSigner })
  l2EndpointContract = await l2EndpointContract.deploy()
  await l2EndpointContract.deployed()

  const endpoint = {
    contract: l2EndpointContract,
    signer: deployerSigner,
  }

  // L2 consumer
  let l2RRConsumerMock = await ethers.getContractFactory('L2RequestResponseConsumerMock', {
    signer: deployerSigner,
  })
  l2RRConsumerMock = await l2RRConsumerMock.deploy(l2EndpointContract.address)
  await l2RRConsumerMock.deployed()

  const consumer = {
    contract: l2RRConsumerMock,
    signer: deployerSigner,
  }

  return {
    endpoint,
    consumer,
    account2,
    account3,
    account4,
    account5,
  }
}

async function requestAndFulfill(requestFn, fulfillFn, dataResponseFn, dataResponse) {
  const { consumer, endpoint } = await loadFixture(deploy)
  const { maxGasLimit: callbackGasLimit } = requestResponseConfig()
  const accMock = 1
  const numSubmission = 1
  const txRequestData = await (
    await consumer.contract[requestFn](accMock, callbackGasLimit, numSubmission)
  ).wait()

  const event = endpoint.contract.interface.parseLog(txRequestData.events[0])
  expect(event.name).to.be.equal('DataRequested')
  const { requestId } = event.args

  await expect(endpoint.contract[fulfillFn](requestId, dataResponse)).revertedWithCustomError(
    endpoint.contract,
    'InvalidSubmitter',
  )

  await (await endpoint.contract.addSubmitter(endpoint.signer.address)).wait()
  await (await endpoint.contract[fulfillFn](requestId, dataResponse)).wait()
  const result = await consumer.contract[dataResponseFn]()
  expect(result).to.be.equal(dataResponse)
}

describe('L2 Request-Response', function () {
  it('Request & Fulfill Uint128', async function () {
    await requestAndFulfill(
      'requestDataUint128',
      'fulfillDataRequestUint128',
      'sResponseUint128',
      1,
    )
  })

  it('Request & Fulfill Int256', async function () {
    await requestAndFulfill('requestDataInt256', 'fulfillDataRequestInt256', 'sResponseInt256', 1)
  })

  it('Request & Fulfill Bool', async function () {
    await requestAndFulfill('requestDataBool', 'fulfillDataRequestBool', 'sResponseBool', true)
  })

  it('Request & Fulfill String', async function () {
    await requestAndFulfill(
      'requestDataString',
      'fulfillDataRequestString',
      'sResponseString',
      'hello',
    )
  })

  it('Request & Fulfill Bytes32', async function () {
    await requestAndFulfill(
      'requestDataBytes32',
      'fulfillDataRequestBytes32',
      'sResponseBytes32',
      ethers.utils.formatBytes32String('hello'),
    )
  })

  it('Request & Fulfill Bytes', async function () {
    await requestAndFulfill(
      'requestDataBytes',
      'fulfillDataRequestBytes',
      'sResponseBytes',
      ethers.utils.formatBytes32String('hello'),
    )
  })
})
