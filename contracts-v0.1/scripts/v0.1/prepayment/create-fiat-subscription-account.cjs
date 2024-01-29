const {
  START_TIME,
  PERIOD,
  REQUEST_NUMBER,
  OWNER_ADDRESS,
  PROVIDER_URL,
  MNEMONIC
} = require('./config.ts')
const { ethers } = require('hardhat')

async function main() {
  const { network, getNamedAccounts } = hre
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

  // Input configuration to create Fiat Account
  const startTime = START_TIME
  const period = PERIOD
  const requestNumber = REQUEST_NUMBER
  const ownerAddress = OWNER_ADDRESS

  console.log(`Input Params:`)
  console.log(`Start time:\t${startTime}`)
  console.log(`Period:\t${period}`)
  console.log(`Request number:\t${requestNumber}`)
  console.log(`Owner address:\t${ownerAddress} \n\n`)

  const txReceipt = await (
    await prepayment.createFiatSubscriptionAccount(startTime, period, requestNumber, ownerAddress)
  ).wait()

  const accId = txReceipt.events[0].args.accId.toString()
  const { balance, reqCount, owner, consumers } = await prepayment.getAccount(accId)
  const [sStartTime, sPeriod, sPeriodReqCount, sSubscriptionPrice] =
    await prepayment.getAccountDetail(accId)

  console.log(`Fiat Subscription Account Created:`)
  console.log(`Account ID:\t${accId}`)
  console.log(`Owner Address:\t${owner}`)
  console.log(`Balance (KLAY):\t${balance}`)
  console.log(`Request Count:\t${reqCount}`)
  console.log(`Consumers:\t${consumers}`)
  console.log(`Subscription Start Time:\t${new Date(sStartTime * 1000).toISOString()}`)
  console.log(`Subscription Duration:\t\t${sPeriod / (60 * 60 * 24)} days`)
  console.log(`Periodic Request Count:\t\t${sPeriodReqCount}`)
  console.log(`Subscription Price (KLAY):\t${sSubscriptionPrice}`)
  console.log('Transaction Hash:\t\t', txReceipt.transactionHash)
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
