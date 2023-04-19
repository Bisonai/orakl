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
  const { deployer, consumer } = await getNamedAccounts()

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
          changeOraclesConfig.addedAdmins,
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

func.id = 'deploy-aggregator'
func.tags = ['aggregator']

module.exports = func
