const path = require('node:path')
const {
  loadJson,
  loadMigration,
  updateMigration,
  validateAggregatorDeployConfig,
  validateAggregatorChangeOraclesConfig
} = require('../../scripts/v0.1/utils.cjs')

const func = async function (hre) {
  const { deployments, getNamedAccounts } = hre
  const { deploy } = deployments
  const { deployer } = await getNamedAccounts()

  console.log('Aggregator.ts')

  const migrationDirPath = `./migration/${network.name}/Aggregator`
  const migrationFilesNames = await loadMigration(migrationDirPath)

  for (const migration of migrationFilesNames) {
    const config = await loadJson(path.join(migrationDirPath, migration))

    let aggregator = undefined

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
          deployConfig.timeout,
          deployConfig.validator,
          deployConfig.decimals,
          deployConfig.description
        ],
        from: deployer,
        log: true
      })
      aggregator = await ethers.getContractAt('Aggregator', aggregatorDeployment.address)

      // AggregatorProxy
      const aggregatorProxyName = `AggregatorProxy_${deployConfig.name}`
      const aggregatorProxyDeployment = await deploy(aggregatorProxyName, {
        contract: 'AggregatorProxy',
        args: [aggregator.address],
        from: deployer,
        log: true
      })

      // DataFeedConsumerMock
      if (['localhost', 'hardhat'].includes(network.name)) {
        await localhostDeployment({
          deploy,
          aggregatorProxyDeployment,
          name: deployConfig.name
        })
      }
    }

    // Update oracles that are allowed to submit to Aggregator /////////////////
    if (config.changeOracles) {
      console.log('changeOracles')
      const changeOraclesConfig = config.changeOracles

      if (!validateAggregatorChangeOraclesConfig(changeOraclesConfig)) {
        throw new Error('Invalid Aggregator changeOracles config')
      }

      aggregator = aggregator
        ? aggregator
        : await ethers.getContractAt('Aggregator', config.aggregatorAddress)

      await (
        await aggregator.changeOracles(
          changeOraclesConfig.removed,
          changeOraclesConfig.added,
          changeOraclesConfig.minSubmissionCount,
          changeOraclesConfig.maxSubmissionCount,
          changeOraclesConfig.restartDelay
        )
      ).wait()
    }

    await updateMigration(migrationDirPath, migration)
  }
}

async function localhostDeployment(args) {
  const { deploy, aggregatorProxyDeployment, name } = args
  const { consumer } = await getNamedAccounts()
  const dataFeedConsumerMockName = `DataFeedConsumerMock_${name}`

  // DataFeedConsumerMock
  await deploy(dataFeedConsumerMockName, {
    contract: 'DataFeedConsumerMock',
    args: [aggregatorProxyDeployment.address],
    from: consumer,
    log: true
  })
}

func.id = 'deploy-aggregator'
func.tags = ['aggregator']

module.exports = func
