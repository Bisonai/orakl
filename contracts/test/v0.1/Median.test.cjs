const { expect } = require('chai')
const { ethers } = require('hardhat')
const { loadFixture } = require('@nomicfoundation/hardhat-network-helpers')
const { median } = require('mathjs')

function floorMedian(arr) {
  return Math.floor(median(arr))
}

async function deploy() {
  let contract = await ethers.getContractFactory('MedianMock', {})
  contract = await contract.deploy()
  await contract.deployed()

  return { contract }
}

describe('Median', function () {
  it('Empty array is not allowed', async function () {
    const { contract } = await loadFixture(deploy)
    const arr = []

    await expect(contract.median(arr)).to.be.reverted
  })

  it('Array with single value => single value', async function () {
    const { contract } = await loadFixture(deploy)
    const arr = [1]

    const res = (await contract.median(arr)).toNumber()
    expect(res).to.be.equal(floorMedian(arr))
  })

  it('Odd number of items => middle value', async function () {
    const { contract } = await loadFixture(deploy)
    const arr = [1, 2, 3]

    const res = (await contract.median(arr)).toNumber()
    expect(res).to.be.equal(floorMedian(arr))
  })

  it('Unsorted numer array of odd length => sort and take middle value', async function () {
    const { contract } = await loadFixture(deploy)
    const arr = [9, 7, 8]

    const res = (await contract.median(arr)).toNumber()
    expect(res).to.be.equal(floorMedian(arr))
  })

  it('Unsorted numer array of even length => sort and take middle value', async function () {
    const { contract } = await loadFixture(deploy)
    const arr = [9, 8, 7, 6]

    const res = (await contract.median(arr)).toNumber()
    expect(res).to.be.equal(floorMedian(arr))
  })

  it('Even number of items => floor of integer average of values around the middle point', async function () {
    const { contract } = await loadFixture(deploy)
    const arr = [1, 2, 3, 4]

    const res = (await contract.median(arr)).toNumber()
    expect(res).to.be.equal(floorMedian(arr))
  })
})
