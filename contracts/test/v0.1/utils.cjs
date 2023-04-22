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
  let falseCount = 0
  let trueCount = 0

  for (let i = 0; i < arr.length; i++) {
    if (arr[i]) {
      trueCount++
    } else {
      falseCount++
    }
  }

  return trueCount >= falseCount
}

module.exports = {
  parseKlay,
  remove0x,
  median,
  majorityVotingBool
}
