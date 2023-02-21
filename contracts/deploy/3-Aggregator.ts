import { HardhatRuntimeEnvironment } from 'hardhat/types'
import { DeployFunction } from 'hardhat-deploy/types'
import { loadJson } from '../scripts/v0.1/utils'
import { IAggregatorConfig } from '../scripts/v0.1/types'

const func: DeployFunction = async function (hre: HardhatRuntimeEnvironment) {
  const { deployments, getNamedAccounts } = hre
  const { deploy } = deployments
  const { deployer, consumer, feedOracle0, feedOracle1, feedOracle2 } = await getNamedAccounts()

  console.log('3-Aggregator.ts')

  const aggregatorConfig: IAggregatorConfig[] = await loadJson(
    `config/${network.name}/aggregator.json`
  )

  // Aggregator
  const config = aggregatorConfig[0]
  const aggregatorDeployment = await deploy('Aggregator', {
    args: [
      config.paymentAmount,
      config.timeout,
      config.validator,
      config.decimals,
      config.description
    ],
    from: deployer,
    log: true
  })

  const aggregator = await ethers.getContractAt('Aggregator', aggregatorDeployment.address)

  // Charge KLAY to Aggregator (used for paying oracles)
  if (config.paymentAmount > 0) {
    const value = ethers.utils.parseEther('1.0')
    await (await aggregator.deposit({ value })).wait()
  }

  // Setup oracles that will contribute to Aggregator
  const removed = []
  const added = config.oracles.length ? config.oracles : [feedOracle0, feedOracle1]
  const addedAdmins = config.admins.length ? config.admins : [feedOracle0, feedOracle1]

  await (
    await aggregator.changeOracles(
      removed,
      added,
      addedAdmins,
      config.minSubmissionCount,
      config.maxSubmissionCount,
      config.restartDelay
    )
  ).wait()

  // Aggregator Proxy
  const aggregatorProxyDeployment = await deploy('AggregatorProxy', {
    args: [aggregatorDeployment.address],
    from: deployer,
    log: true
  })

  if (['localhost', 'hardhat'].includes(network.name)) {
    await localhostDeployment({ deploy, consumer, aggregatorProxyDeployment })
  }
}

async function localhostDeployment(args) {
  const { deploy, consumer, aggregatorProxyDeployment } = args

  // Data feed consumer
  await deploy('DataFeedConsumerMock', {
    args: [aggregatorProxyDeployment.address],
    from: consumer,
    log: true
  })
}

export default func
func.id = 'deploy-aggregator'
func.tags = ['aggregator']
