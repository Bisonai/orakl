import { expect } from 'chai'
import pkg from 'hardhat'
const { ethers } = pkg

function preprocessArray(a) {
  if (a.length % 2 == 0) {
    a.push(a[a.length - 1])
  }
  return a
}

function getMiddleIndex(a) {
  return Math.round(a.length / 2)
}

describe('QuickSelect', function () {
  let contract

  beforeEach(async function () {
    contract = await ethers.getContractFactory('QuickSelectMock')
    contract = await contract.deploy()
    await contract.deployed()
  })

  it('Compute median on sorted array of even length', async function () {
    const a = preprocessArray([1, 2])
    const k = getMiddleIndex(a)
    const median = await contract.quickSelect(a, k)
    expect(Number(median)).to.be.equal(2)
  })

  it('Compute median on sorted array of odd length', async function () {
    const a = preprocessArray([1, 2, 3])
    const k = getMiddleIndex(a)
    const median = await contract.quickSelect(a, k)
    expect(Number(median)).to.be.equal(2)
  })

  it('Compute median on UNsorted array of even length', async function () {
    const a = preprocessArray([2, 1])
    const k = getMiddleIndex(a)
    const median = await contract.quickSelect(a, k)
    expect(Number(median)).to.be.equal(1)
  })

  it('Compute median on UNsorted array of odd length', async function () {
    const a = preprocessArray([3, 2, 1])
    const k = getMiddleIndex(a)
    const median = await contract.quickSelect(a, k)
    expect(Number(median)).to.be.equal(2)
  })
})
