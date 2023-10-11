const { FEE_RATIO, ACC_ID, PROVIDER_URL, MNEMONIC } = require('./config.ts')
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

  // Input configuration to create Fiat Account
  const accId = ACC_ID
  const feeRatio = FEE_RATIO

  console.log(`Input Params:`)
  console.log(`Account Id:\t${accId}`)
  console.log(`Fee ratio:\t${feeRatio}\n\n`)

  const txReceipt = await (await prepayment.setFeeRatio(accId, feeRatio)).wait()

  const { balance, reqCount, owner, consumers } = await prepayment.getAccount(accId)
  const sFeeRatio = await prepayment.getFeeRatio(accId)

  console.log(`Klay Discount Account Created:`)
  console.log(`Account ID:\t${accId}`)
  console.log(`Owner Address:\t${owner}`)
  console.log(`Balance (KLAY):\t${balance}`)
  console.log(`Request Count:\t${reqCount}`)
  console.log(`Consumers:\t${consumers}`)
  console.log(`Fee Ratio:\t${sFeeRatio / 100} %`)
  console.log('Transaction Hash:', txReceipt.transactionHash)
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
