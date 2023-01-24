import { ethers } from 'hardhat'
import hre from 'hardhat'

async function main() {
  const { network } = hre
  const { feedOracle0 } = await hre.getNamedAccounts()

  if (network.name == 'localhost') {
    console.log('Exiting')
    return
  }

  const aggregator = await ethers.getContract('Aggregator')

  const removed = [feedOracle0]
  const added = [
    '0x96fD7c07A965dD4c32cda4B9268D86436E57c5e5',
    '0x3f27dcd626Ebc3Ad6a9b3A3b828352345a76c50C'
  ]
  const addedAdmins = added
  const minSubmissionCount = 1
  const maxSubmissionCount = 2
  const restartDelay = 0

  await (
    await aggregator.changeOracles(
      removed,
      added,
      addedAdmins,
      minSubmissionCount,
      maxSubmissionCount,
      restartDelay
    )
  ).wait()
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
