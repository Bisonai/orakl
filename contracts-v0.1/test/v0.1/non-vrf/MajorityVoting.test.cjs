const { expect } = require('chai')
const { ethers } = require('hardhat')
const { loadFixture } = require('@nomicfoundation/hardhat-network-helpers')

async function deploy() {
  let contract = await ethers.getContractFactory('MajorityVotingMock', {})
  contract = await contract.deploy()
  await contract.deployed()

  return { contract }
}

describe('MajorityVoting', function () {
  it('Single true value', async function () {
    const { contract } = await loadFixture(deploy)
    const arr = [true]
    const res = true
    expect(await contract.voting(arr)).to.be.equal(res)
  })

  it('Single false value', async function () {
    const { contract } = await loadFixture(deploy)
    const arr = [false]
    const res = false
    expect(await contract.voting(arr)).to.be.equal(res)
  })

  it('List cannot be of even length', async function () {
    const { contract } = await loadFixture(deploy)
    const arr = [true, false]
    await expect(contract.voting(arr)).to.be.revertedWithCustomError(contract, 'EvenLengthList')
  })

  it('True majority', async function () {
    const { contract } = await loadFixture(deploy)
    const arr = [true, false, true]
    const res = true
    expect(await contract.voting(arr)).to.be.equal(res)
  })

  it('False majority', async function () {
    const { contract } = await loadFixture(deploy)
    const arr = [true, false, false]
    const res = false
    expect(await contract.voting(arr)).to.be.equal(res)
  })
})
