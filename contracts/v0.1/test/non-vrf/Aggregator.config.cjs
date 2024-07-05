function aggregatorConfig() {
  const timeout = 10
  const validator = ethers.constants.AddressZero // no validator
  const decimals = 18
  const description = 'Test Aggregator'

  return {
    timeout,
    validator,
    decimals,
    description,
  }
}

module.exports = {
  aggregatorConfig,
}
