async function setMinBalance(coordinatorContract, minBalance) {
  await coordinatorContract.setMinBalance(ethers.utils.parseUnits(minBalance, 'ether'))
}

module.exports = {
  setMinBalance
}
