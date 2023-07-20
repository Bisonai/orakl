const { expect } = require('chai')
const { ethers } = require('hardhat')
const { loadFixture } = require('@nomicfoundation/hardhat-network-helpers')
const {
  deploy: deployRegistry,
  propose,
  confirm,
  setProposeFee,
  withdraw
} = require('./Registry.utils.cjs')
const { parseKlay, getBalance, createSigners } = require('./utils.cjs')
const { exp } = require('mathjs')

async function deploy() {
  const {
    account0: deployerSigner,
    account1,
    account2,
    account3,
    account4,
    account5
  } = await createSigners()

  const registryContract = await deployRegistry(deployerSigner)

  return {
    deployerSigner,
    account1,
    account2,
    account3,
    account4,
    account5,
    registryContract
  }
}
describe('Registry', function () {
  it('set fee', async function () {
    const fee = parseKlay(1)
    const { registryContract, deployerSigner } = await loadFixture(deploy)
    const newFee = await setProposeFee(registryContract, deployerSigner, fee)
    expect(newFee).to.be.equal(fee)
  })

  it('propose & confirm', async function () {
    const { registryContract, deployerSigner, account1, account2, account3 } = await loadFixture(
      deploy
    )
    const fee = parseKlay(1)
    const pChainID = '100001'
    const jsonRpc = 'https://123'
    const endpoint = account1.address
    const l1Aggregator = account2.address
    const l2Aggregator = account3.address
    const { chainID } = await propose(
      registryContract,
      deployerSigner,
      pChainID,
      jsonRpc,
      endpoint,
      l1Aggregator,
      l2Aggregator,
      fee
    )
    expect(chainID).to.be.equal(pChainID)

    const data = await confirm(registryContract, deployerSigner, pChainID)
    expect(data.chainID).to.be.equal(pChainID)
  })

  it('withdraw', async function () {
    const { registryContract, deployerSigner, account1, account2, account3 } = await loadFixture(
      deploy
    )
    const fee = parseKlay(1)
    const pChainID = '100001'
    const jsonRpc = 'https://123'
    const endpoint = account1.address
    const l1Aggregator = account2.address
    const l2Aggregator = account3.address
    const { chainID } = await propose(
      registryContract,
      deployerSigner,
      pChainID,
      jsonRpc,
      endpoint,
      l1Aggregator,
      l2Aggregator,
      fee
    )
    const beforeWithdraw = await getBalance(registryContract.address)
    await withdraw(registryContract, deployerSigner, fee)
    const afterWithdraw = await getBalance(registryContract.address)

    expect(beforeWithdraw).to.be.equal(afterWithdraw + fee)
  })
})
