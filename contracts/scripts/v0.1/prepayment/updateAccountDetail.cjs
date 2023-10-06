const { ethers } = require('hardhat')
const hre = require('hardhat')

async function main() {
  const { network } = hre
  let _deployerSigner

  if (network.name == 'localhost') {
    const { deployer } = await hre.getNamedAccounts()
    _deployerSigner = deployer
  } else {
    const PROVIDER = process.env.PROVIDER
    const MNEMONIC = process.env.MNEMONIC || ''
    const provider = new ethers.providers.JsonRpcProvider(PROVIDER)
    _deployerSigner = ethers.Wallet.fromMnemonic(MNEMONIC).connect(provider)
  }

  // Get the Prepayment contract instance
  let prepayment = await ethers.getContract('Prepayment')
  prepayment = await ethers.getContractAt('Prepayment', prepayment.address, _deployerSigner)

  // Input configuration to create Fiat Account
  const accId = 1
  const startTime = Math.round(new Date().getTime() / 1000) - 60 * 60 // Start time in seconds
  const period = 60 * 60 * 24 * 7 // Duration in seconds
  const requestNumber = 20 // Number of requests
  const subscriptionPrice = ethers.utils.parseEther('1') // SubscriptionPrice in Klay

  const txReceipt = await (
    await prepayment.updateAccountDetail(accId, startTime, period, requestNumber, subscriptionPrice)
  ).wait()

  const { balance, reqCount, owner, consumers } = await prepayment.getAccount(accId)
  const [sStartTime, sPeriod, sPeriodReqCount, sSubscriptionPrice] =
    await prepayment.getAccountDetail(accId)

  console.log(`Account Details after update:`)
  console.log(`Account ID:\t${accId}`)
  console.log(`Owner Address:\t${owner}`)
  console.log(`Balance (KLAY):\t${balance}`)
  console.log(`Request Count:\t${reqCount}`)
  console.log(`Consumers:\t${consumers}`)
  console.log(`Subscription Start Time:\t${new Date(sStartTime * 1000).toISOString()}`)
  console.log(`Subscription Duration:\t\t${sPeriod / (60 * 60 * 24)} days`)
  console.log(`Periodic Request Count:\t\t${sPeriodReqCount}`)
  console.log(
    `Subscription Price (KLAY):\t${ethers.utils.formatUnits(sSubscriptionPrice, 'ether')}`
  )
  console.log('Transaction Hash:\t\t', txReceipt.transactionHash)
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
