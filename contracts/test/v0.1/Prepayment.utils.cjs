async function createAccount(prepayment, signer) {
  const tx = await (await prepayment.connect(signer).createAccount()).wait()
  const event = prepayment.interface.parseLog(tx.events[0])
  const { accId } = event.args
  return accId
}

async function deposit(prepayment, signer, accId, amount) {
  await prepayment.connect(signer).deposit(accId, {
    value: ethers.utils.parseUnits(amount, 'ether')
  })
}

module.exports = {
  createAccount,
  deposit
}
