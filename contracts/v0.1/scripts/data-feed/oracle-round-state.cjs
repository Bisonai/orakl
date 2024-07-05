import hre, { ethers } from 'hardhat'

async function main() {
  const { network } = hre
  let _consumer
  let _feedOracle0

  if (network.name == 'localhost') {
    const { consumer, feedOracle0 } = await hre.getNamedAccounts()
    _consumer = consumer
    _feedOracle0 = feedOracle0
  } else {
    const { feedOracle0 } = await hre.getNamedAccounts()
    _feedOracle0 = feedOracle0
    const PROVIDER = process.env.PROVIDER
    const MNEMONIC = process.env.MNEMONIC || ''
    const provider = new ethers.providers.JsonRpcProvider(PROVIDER)
    _consumer = ethers.Wallet.fromMnemonic(MNEMONIC).connect(provider)
  }

  let aggregator = await ethers.getContract('Aggregator')
  aggregator = await ethers.getContractAt('Aggregator', aggregator.address)

  console.log('Aggregator', aggregator.address)

  const {
    _eligibleToSubmit,
    _roundId,
    _latestSubmission,
    _startedAt,
    _timeout,
    _availableFunds,
    _oracleCount,
    _paymentAmount,
  } = await aggregator.connect(_consumer).oracleRoundState(_feedOracle0, 0)

  console.log(`_eligibleToSubmit  ${_eligibleToSubmit}`)
  console.log(`_roundId           ${_roundId}`)
  console.log(`_latestSubmission  ${_latestSubmission}`)
  console.log(`_startedAt         ${_startedAt}`)
  console.log(`_timeout           ${_timeout}`)
  console.log(`_availableFunds    ${_availableFunds}`)
  console.log(`_oracleCount       ${_oracleCount}`)
  console.log(`_paymentAmount     ${_paymentAmount}`)
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
