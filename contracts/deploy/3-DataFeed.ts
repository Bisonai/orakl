import { HardhatRuntimeEnvironment } from 'hardhat/types'
import { DeployFunction } from 'hardhat-deploy/types'
const dataFeedConfig = require('../config/data-feed.json')

const func: DeployFunction = async function (hre: HardhatRuntimeEnvironment) {
  const { deployments, getNamedAccounts } = hre
  const { deploy } = deployments
  const { deployer, consumer, feedOracle0, feedOracle1, feedOracle2 } = await getNamedAccounts()

  console.log('3-DataFeed.ts')

  // Aggregator
  const config = dataFeedConfig['KLAY/USD']
  const aggregatorDeployment = await deploy('Aggregator', {
    args: [
      config.paymentAmount,
      config.timeout,
      config.validator,
      config.minSubmissionValue,
      config.maxSubmissionValue,
      config.decimals,
      config.description
    ],
    from: deployer,
    log: true
  })

  const aggregator = await ethers.getContractAt('Aggregator', aggregatorDeployment.address)

  // Charge KLAY to Aggregator (usedd for paying oracles)
  const value = ethers.utils.parseEther('1.0')
  await aggregator.deposit({ value })

  // Setup oracles that will contribute to Aggregator
  const removed = []
  const added = [feedOracle0]
  // FIXME Most likely wrong. Learn more about addedAdmins.
  const addedAdmins = [feedOracle0]

  await aggregator.changeOracles(
    removed,
    added,
    addedAdmins,
    config.minSubmissionCount,
    config.maxSubmissionCount,
    config.restartDelay
  )

  // Aggregator Proxy
  const aggregatorProxyDeployment = await deploy('AggregatorProxy', {
    args: [aggregatorDeployment.address],
    from: deployer,
    log: true
  })

  // Data feed consumer
  await deploy('DataFeedConsumerMock', {
    args: [aggregatorProxyDeployment.address],
    from: consumer,
    log: true
  })
}

export default func
func.id = 'deploy-data-feed'
func.tags = ['data-feed']
