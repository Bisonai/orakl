import { HardhatRuntimeEnvironment } from 'hardhat/types'
import { DeployFunction } from 'hardhat-deploy/types'
import * as path from 'node:path'
import {
  loadJson,
  loadMigration,
  updateMigration,
  validateAggregatorDeployConfig,
  validateAggregatorChangeOraclesConfig
} from '../../scripts/v0.1/utils'
import { IAggregatorConfig } from '../../scripts/v0.1/types'

const func: DeployFunction = async function (hre: HardhatRuntimeEnvironment) {
  const { deployments, getNamedAccounts } = hre
  const { deploy } = deployments
  const { deployer, consumer } = await getNamedAccounts()

  console.log('Aggregator.ts')

  const migrationDirPath = `./migration/${network.name}/Aggregator`
  const migrationFilesNames = await loadMigration(migrationDirPath)

  for (const migration of migrationFilesNames) {
    const config: IAggregatorConfig = await loadJson(path.join(migrationDirPath, migration))

    let aggregator: ethers.Contract = undefined

    // Deploy Aggregator and AggregatorProxy ////////////////////////////////////
    if (config.deploy) {
      const deployConfig = config.deploy
      if (!validateAggregatorDeployConfig(deployConfig)) {
        throw new Error('Invalid Aggregator deploy config')
      }

      // Aggregator
      const aggregatorName = `Aggregator_${deployConfig.name}`
      const aggregatorDeployment = await deploy(aggregatorName, {
        contract: 'Aggregator',
        args: [
          deployConfig.paymentAmount,
          deployConfig.timeout,
          deployConfig.validator,
          deployConfig.decimals,
          deployConfig.description
        ],
        from: deployer,
        log: true
      })

      // AggregatorProxy
      const aggregatorProxyName = `AggregatorProxy_${deployConfig.name}`
      const aggregatorProxyDeployment = await deploy(aggregatorProxyName, {
        contract: 'AggregatorProxy',
        args: [aggregatorDeployment.address],
        from: deployer,
        log: true
      })

      // Deposit KLAY to Aggregator (used for paying oracles)
      aggregator = await ethers.getContractAt('Aggregator', aggregatorDeployment.address)
      if (config.paymentAmount > 0) {
        const value = ethers.utils.parseEther(deployConfig.depositAmount)
        await (await aggregator.deposit({ value })).wait()
      }

      // DataFeedConsumerMock
      if (['localhost', 'hardhat'].includes(network.name)) {
        await localhostDeployment({
          deploy,
          consumer,
          aggregatorProxyDeployment,
          name: deployConfig.name
        })
      }
    }

    // Update oracles that are allowed to submit to Aggregator /////////////////
    if (config.changeOracles) {
      const changeOraclesConfig = config.changeOracles
      aggregator = aggregator
        ? aggregator
        : await ethers.getContractAt('Aggregator', changeOraclesConfig.aggregatorAddress)

      await (
        await aggregator.changeOracles(
          changeOraclesConfig.removed,
          changeOraclesConfig.added,
          changeOraclesConfig.addedAdmins,
          changeOraclesConfig.minSubmissionCount,
          changeOraclesConfig.maxSubmissionCount,
          changeOraclesConfig.restartDelay
        )
      ).wait()
    }

    await updateMigration(migrationDirPath, migration)
  }

  return 0
}

async function localhostDeployment(args) {
  const { deploy, consumer, aggregatorProxyDeployment, name } = args
  const dataFeedConsumerMockName = `DataFeedConsumerMock_${name}`

  // DataFeedConsumerMock
  await deploy(dataFeedConsumerMockName, {
    contract: 'DataFeedConsumerMock',
    args: [aggregatorProxyDeployment.address],
    from: consumer,
    log: true
  })
}

export default func
func.id = 'deploy-aggregator'
func.tags = ['aggregator']
