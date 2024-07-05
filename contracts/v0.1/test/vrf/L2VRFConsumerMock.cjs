const { expect } = require('chai')
const { ethers } = require('hardhat')
const { time, loadFixture } = require('@nomicfoundation/hardhat-network-helpers')

const { createSigners } = require('../utils.cjs')
const { vrfConfig } = require('./VRFCoordinator.config.cjs')

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
  let l2VRFConsumerMock = await ethers.getContractFactory('L2VRFConsumerMock', {
    signer: deployerSigner,
  })
  l2VRFConsumerMock = await l2VRFConsumerMock.deploy(l2EndpointContract.address)
  await l2VRFConsumerMock.deployed()

  const consumer = {
    contract: l2VRFConsumerMock,
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

describe('Consumer', function () {
  it('Request and fullfil', async function () {
    const { consumer, endpoint } = await loadFixture(deploy)
    const { maxGasLimit: callbackGasLimit, keyHash } = vrfConfig()
    const accMock = 1
    const numWords = 1
    const txRequestRandomWords = await (
      await consumer.contract.requestRandomWords(keyHash, accMock, callbackGasLimit, numWords)
    ).wait()
    const event = endpoint.contract.interface.parseLog(txRequestRandomWords.events[0])
    expect(event.name).to.be.equal('RandomWordsRequested')
    const { requestId } = event.args

    const randomWords = [1]
    await expect(
      endpoint.contract.fulfillRandomWords(requestId, randomWords),
    ).revertedWithCustomError(endpoint.contract, 'InvalidSubmitter')

    await (await endpoint.contract.addSubmitter(endpoint.signer.address)).wait()
    await (await endpoint.contract.fulfillRandomWords(requestId, randomWords)).wait()
    const result = await consumer.contract.sRandomWord()
    expect(result).to.be.equal((randomWords % 50) + 1)
  })
})
