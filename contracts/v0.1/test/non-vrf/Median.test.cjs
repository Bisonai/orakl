const { expect } = require('chai')
const { ethers } = require('hardhat')
const { loadFixture } = require('@nomicfoundation/hardhat-network-helpers')
const { median } = require('../utils.cjs')

function medianBN(arr) {
  const arrBN = arr.map((x) => ethers.BigNumber.from(x))
  arrBN.sort((a, b) => {
    const A = a._hex.toUpperCase()
    const B = b._hex.toUpperCase()

    if (A.length < B.length) {
      return -1
    } else if (A.length > B.length) {
      return 1
    } else {
      if (A < B) {
        return -1
      } else if (A > B) {
        return 1
      }
      return 0
    }
  })

  const pivot = Math.floor(arrBN.length / 2)
  if (arrBN.length % 2 == 0) {
    return arrBN[pivot - 1].add(arrBN[pivot]).div(2)
  } else {
    return arrBN[pivot]
  }
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
    expect(res).to.be.equal(median(arr))
  })

  it('Odd number of items => middle value', async function () {
    const { contract } = await loadFixture(deploy)
    const arr = [1, 2, 3]

    const res = (await contract.median(arr)).toNumber()
    expect(res).to.be.equal(median(arr))
  })

  it('Unsorted numer array of odd length => sort and take middle value', async function () {
    const { contract } = await loadFixture(deploy)
    const arr = [9, 7, 8]

    const res = (await contract.median(arr)).toNumber()
    expect(res).to.be.equal(median(arr))
  })

  it('Unsorted numer array of even length => sort and take middle value', async function () {
    const { contract } = await loadFixture(deploy)
    const arr = [9, 8, 7, 6]

    const res = (await contract.median(arr)).toNumber()
    expect(res).to.be.equal(median(arr))
  })

  it('Even number of items => floor of integer average of values around the middle point', async function () {
    const { contract } = await loadFixture(deploy)
    const arr = [1, 2, 3, 4]

    const res = (await contract.median(arr)).toNumber()
    expect(res).to.be.equal(median(arr))
  })

  it('Large array of numbers', async function () {
    // Median is computed different for arrays with lenght larger than 7
    const { contract } = await loadFixture(deploy)
    const arr = [
      ['1', '2', '3', '4', '5', '6', '7', '8'],
      ['1', '2', '3', '4', '5', '6', '7', '8', '1', '2', '3', '4', '5', '6', '7', '8'],
      ['876243876324', '87923493492782', '97832497234908234978324', '9723497234908234023'],
      ['876243876324', '87923493492782', '9723497234908234023'],
      [
        '987234873429872359873252343',
        '982345897325867324175624194',
        '890129821971482484642397823',
        '192978387212138712981287233',
      ],
      [
        '987234873429872359873252342987234873429872359873252342',
        '982345897325867324175624194987234873429872359873252343',
        '890129821971482484642397823982345897325867324175624194',
        '192978387212138712981278923890129821971482484642397823',
        '328923487231897219081298783192978387212138712981287233',
      ],
    ]

    for (const A of arr) {
      expect(await contract.median(A)).to.be.equal(medianBN(A))
    }
  })
})
