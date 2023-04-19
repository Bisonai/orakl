function parseKlay(amount) {
  return ethers.utils.parseUnits(amount.toString(), 18)
}

function remove0x(s) {
  if (s.substring(0, 2) == '0x') {
    return s.substring(2)
  }
}

module.exports = {
  parseKlay,
  remove0x
}
