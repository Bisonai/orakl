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
  const feeRatio = 10000 // 100%
  const ownerAddress = '0x30E30C3B6313FF232E93593b883fC8A8AF8BB627'
  console.log(`Input Params:`)
  console.log(`fee ratio:\t${feeRatio}`)
  console.log(`Owner Address:\t${ownerAddress}\n\n`)

  const txReceipt = await (
    await prepayment.createKlayDiscountAccount(feeRatio, ownerAddress)
  ).wait()

  const accId = txReceipt.events[0].args.accId.toString()
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
