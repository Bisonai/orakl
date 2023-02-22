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

  const removed = []
  const added = []
  const addedAdmins = added
  const minSubmissionCount = 1
  const maxSubmissionCount = 1
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
