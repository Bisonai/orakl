import { HardhatRuntimeEnvironment } from 'hardhat/types'
import { DeployFunction } from 'hardhat-deploy/types'
const dataFeedConfig = require('../config/data-feed.json')

const func: DeployFunction = async function (hre: HardhatRuntimeEnvironment) {
  const { deployments, getNamedAccounts } = hre
  const { deploy } = deployments
  const { deployer, consumer, feedOracle0, feedOracle1, feedOracle2 } = await getNamedAccounts()

  console.log('3-Aggregator.ts')

  // Aggregator
  const config = dataFeedConfig['KLAY/USD']
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

  // Charge KLAY to Aggregator (usedd for paying oracles)
  const value = ethers.utils.parseEther('1.0')
  await (await aggregator.deposit({ value })).wait()

  // Setup oracles that will contribute to Aggregator
  const removed = []
  const added = [feedOracle0]
  // FIXME Most likely wrong. Learn more about addedAdmins.
  const addedAdmins = [feedOracle0]

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
