const { median: medianFn } = require('mathjs')

async function getBalance(address) {
  return await ethers.provider.getBalance(address)
}

function parseKlay(amount) {
  return ethers.utils.parseUnits(amount.toString(), 18)
}

function remove0x(s) {
  if (s.substring(0, 2) == '0x') {
    return s.substring(2)
  }
}

function median(arr) {
  return Math.floor(medianFn(arr))
}

function majorityVotingBool(arr) {
  const trueCount = arr.reduce((acc, x) => acc + x, 0)
  const falseCount = arr.length - trueCount
  return trueCount >= falseCount
}

async function createSigners() {
  let { account0, account1, account2, account3, account4, account5, account6, account7, account8 } =
    await hre.getNamedAccounts()

  account0 = await ethers.getSigner(account0)
  account1 = await ethers.getSigner(account1)
  account2 = await ethers.getSigner(account2)
  account3 = await ethers.getSigner(account3)
  account4 = await ethers.getSigner(account4)
  account5 = await ethers.getSigner(account5)
  account6 = await ethers.getSigner(account6)
  account7 = await ethers.getSigner(account7)
  account8 = await ethers.getSigner(account8)

  return {
    account0,
    account1,
    account2,
    account3,
    account4,
    account5,
    account6,
    account7,
    account8,
  }
}

module.exports = {
  getBalance,
  parseKlay,
  remove0x,
  median,
  majorityVotingBool,
  createSigners,
}
