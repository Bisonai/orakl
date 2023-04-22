const { median: medianFn } = require('mathjs')

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

module.exports = {
  parseKlay,
  remove0x,
  median,
  majorityVotingBool
}
