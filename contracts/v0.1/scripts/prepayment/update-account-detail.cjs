const {
  ACC_ID,
  PERIOD,
  REQUEST_NUMBER,
  START_TIME,
  SUBSCRIPTION_PRICE,
  PROVIDER_URL,
  MNEMONIC,
} = require('./config.ts')
const { ethers } = require('hardhat')

async function main() {
  const { network } = hre
  let _deployerSigner

  if (network.name == 'localhost') {
    const { deployer } = await hre.getNamedAccounts()
    _deployerSigner = deployer
  } else {
    const provider = new ethers.providers.JsonRpcProvider(PROVIDER_URL)
    _deployerSigner = ethers.Wallet.fromMnemonic(MNEMONIC).connect(provider)
  }

  // Get the Prepayment contract
  let prepayment = await ethers.getContract('Prepayment')
  prepayment = await ethers.getContractAt('Prepayment', prepayment.address, _deployerSigner)

  // Input configuration to update account details
  const accId = ACC_ID
  const startTime = START_TIME
  const period = PERIOD
  const requestNumber = REQUEST_NUMBER
  const subscriptionPrice = SUBSCRIPTION_PRICE

  console.log(`Input Params:`)
  console.log(`Account Id:\t${accId}`)
  console.log(`Start time:\t${startTime}`)
  console.log(`Period:\t${period}`)
  console.log(`Request number:\t${requestNumber}`)
  console.log(`Subscription price:\t${subscriptionPrice}\n\n`)

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
    `Subscription Price (KLAY):\t${ethers.utils.formatUnits(sSubscriptionPrice, 'ether')}`,
  )
  console.log('Transaction Hash:\t\t', txReceipt.transactionHash)
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
