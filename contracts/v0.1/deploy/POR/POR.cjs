const path = require('node:path')
const { expect } = require('chai')
const {
  loadJson,
  loadMigration,
  updateMigration,
  validateAggregatorDeployConfig,
  validateAggregatorChangeOraclesConfig,
  validateAggregatorRedirectProxyConfig,
  getFormattedDate,
} = require('../../scripts/utils.cjs')

const func = async function (hre) {
  const { deployments, getNamedAccounts } = hre
  const { deploy } = deployments
  const { deployer } = await getNamedAccounts()

  console.log('POR.cjs')

  const migrationDirPath = `./migration/${network.name}/POR`
  const migrationFilesNames = await loadMigration(migrationDirPath)
  const date = getFormattedDate()

  for (const migration of migrationFilesNames) {
    const config = await loadJson(path.join(migrationDirPath, migration))

    let aggregator = undefined

    // Deploy POR ////////////////////////////////////
    if (config.deploy) {
      const deployConfig = config.deploy
      if (!validateAggregatorDeployConfig(deployConfig)) {
        throw new Error('Invalid POR deploy config')
      }

      // Aggregator
      const aggregatorName = `POR_${deployConfig.name}_${date}`
      const aggregatorDeployment = await deploy(aggregatorName, {
        contract: 'Aggregator',
        args: [
          deployConfig.timeout,
          deployConfig.validator,
          deployConfig.decimals,
          deployConfig.description,
        ],
        from: deployer,
        log: true,
      })
      aggregator = await ethers.getContractAt('Aggregator', aggregatorDeployment.address)
    }

    // Update oracles that are allowed to submit to Aggregator /////////////////
    if (config.changeOracles) {
      console.log('changeOracles')
      const changeOraclesConfig = config.changeOracles

      if (!validateAggregatorChangeOraclesConfig(changeOraclesConfig)) {
        throw new Error('Invalid POR changeOracles config')
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
          changeOraclesConfig.restartDelay,
        )
      ).wait()
    }

    // redirect Proxy ////////////////////////////////////
    if (config.redirectProxy) {
      console.log('Redirect Proxy')
      const redirectProxyConfig = config.redirectProxy
      if (!validateAggregatorRedirectProxyConfig(redirectProxyConfig)) {
        throw new Error('Invalid POR Redirect Proxy config')
      }

      const proxy = await ethers.getContractAt('AggregatorProxy', redirectProxyConfig.proxyAddress)
      const aggregatorAddress = redirectProxyConfig.aggregator
      const proposedAggregator = aggregator
        ? aggregator.address
        : redirectProxyConfig.proposedAggregator

      if (redirectProxyConfig.status == 'initial') {
        // Propose new POR contracts address
        console.log('Initial Stage')
        expect(await proxy.aggregator()).to.be.eq(aggregatorAddress)
        await (await proxy.proposeAggregator(proposedAggregator)).wait()

        const currentProposedAggregator = await proxy.proposedAggregator()
        expect(currentProposedAggregator).to.be.eq(proposedAggregator)

        console.log(`Proposed proxy aggregator:${proposedAggregator}`)
      } else if (redirectProxyConfig.status == 'confirm') {
        // Confirm new POR contract address
        console.log('Confirming Proxy')
        expect(await proxy.aggregator()).to.be.eq(aggregatorAddress)
        expect(await proxy.proposedAggregator()).to.be.eq(proposedAggregator)
        await (await proxy.confirmAggregator(proposedAggregator)).wait()

        const confirmedAggregator = await proxy.aggregator()
        expect(confirmedAggregator).to.be.eq(proposedAggregator)

        console.log(
          `Proxy POR address redirected from ${aggregatorAddress} to new ${confirmedAggregator}`,
        )
      } else if (redirectProxyConfig.status == 'revert') {
        // Revert back to old POR contract address
        console.log('Revert Proxy')
        expect(await proxy.aggregator()).to.be.eq(proposedAggregator)
        await (await proxy.proposeAggregator(aggregatorAddress)).wait()
        await (await proxy.confirmAggregator(aggregatorAddress)).wait()
        const revertedAggregator = await proxy.aggregator()
        expect(revertedAggregator).to.be.eq(aggregatorAddress)

        console.log(
          `Proxy POR address reverted from ${proposedAggregator} to ${revertedAggregator}`,
        )
      } else {
        console.log('Wrong proxyRedirect method')
      }
    } else if (config.deploy) {
      // Deploy PORProxy ////////////////////////////////////
      const deployConfig = config.deploy
      const aggregatorProxyName = `PORProxy_${deployConfig.name}_${date}`
      const aggregatorProxyDeployment = await deploy(aggregatorProxyName, {
        contract: 'AggregatorProxy',
        args: [aggregator.address],
        from: deployer,
        log: true,
      })

      // DataFeedConsumerMock
      if (['localhost', 'hardhat'].includes(network.name)) {
        await localhostDeployment({
          deploy,
          aggregatorProxyDeployment,
          name: deployConfig.name,
        })
      }
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
    log: true,
  })
}

func.id = 'deploy-POR'
func.tags = ['POR']

module.exports = func
