const { expect } = require('chai')
const { ethers } = require('hardhat')
const { loadFixture } = require('@nomicfoundation/hardhat-network-helpers')

async function deploy() {
  let contract = await ethers.getContractFactory('ConversionTest')
  contract = await contract.deploy()
  await contract.deployed()

  return contract
}

describe('Conversion', function () {
  it('uint128 -> int256', async function () {
    const contract = await loadFixture(deploy)
    await contract.uint128ToInt256Test()
  })

  it('int256 -> uint128', async function () {
    const contract = await loadFixture(deploy)
    await contract.int256ToUint128Test()
  })
})
