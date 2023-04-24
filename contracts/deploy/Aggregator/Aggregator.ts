import { HardhatRuntimeEnvironment } from 'hardhat/types'
import { DeployFunction } from 'hardhat-deploy/types'
import { expect } from 'chai'
import * as path from 'node:path'
import {
  loadJson,
  loadMigration,
  updateMigration,
  validateAggregatorDeployConfig,
  validateAggregatorChangeOraclesConfig,
  validateAggregatorRedirectProxyConfig
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

    // Deploy Aggregator ////////////////////////////////////
    if (config.deploy) {
      const deployConfig = config.deploy
      if (!validateAggregatorDeployConfig(deployConfig)) {
        throw new Error('Invalid Aggregator deploy config')
      }

      // Aggregator
      const now = new Date().getTime()
      const aggregatorName = `Aggregator_${deployConfig.name}_${now}`
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

      // Deposit KLAY to Aggregator (used for paying oracles)
      aggregator = await ethers.getContractAt('Aggregator', aggregatorDeployment.address)
      if (config.paymentAmount > 0) {
        const value = ethers.utils.parseEther(deployConfig.depositAmount)
        await (await aggregator.deposit({ value })).wait()
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
    }

    // redirect Proxy ////////////////////////////////////
    if (config.redirectProxy) {
      console.log('Redirect Proxy')
      const redirectProxyConfig = config.redirectProxy
      if (!validateAggregatorRedirectProxyConfig(redirectProxyConfig)) {
        throw new Error('Invalid Aggregator Redirect Proxy config')
      }

      const proxy = await ethers.getContractAt('AggregatorProxy', redirectProxyConfig.proxyAddress)
      const aggregatorAddress = redirectProxyConfig.aggregator
      const proposedAggregator = aggregator
        ? aggregator.address
        : redirectProxyConfig.proposedAggregator

      if (redirectProxyConfig.status == 'initial') {
        // Propose new aggregator
        console.log('Initial Stage')
        expect(await proxy.aggregator()).to.be.eq(aggregatorAddress)
        await (await proxy.proposeAggregator(proposedAggregator)).wait()

        const currentProposedAggregator = await proxy.proposedAggregator()
        expect(currentProposedAggregator).to.be.eq(proposedAggregator)

        console.log(`Proposed proxy aggregator:${proposedAggregator}`)
      } else if (redirectProxyConfig.status == 'confirm') {
        // Confirm new aggregator from Proxy
        console.log('Confirming Proxy')
        expect(await proxy.aggregator()).to.be.eq(aggregatorAddress)
        expect(await proxy.proposedAggregator()).to.be.eq(proposedAggregator)
        await (await proxy.confirmAggregator(proposedAggregator)).wait()

        const confirmedAggregator = await proxy.aggregator()
        expect(confirmedAggregator).to.be.eq(proposedAggregator)

        console.log(
          `Proxy Aggregator redirected from ${aggregatorAddress} to new ${confirmedAggregator}`
        )
      } else if (redirectProxyConfig.status == 'revert') {
        // Revert back to old Aggregator Address
        console.log('Revert Proxy')
        expect(await proxy.aggregator()).to.be.eq(proposedAggregator)
        await (await proxy.proposeAggregator(aggregatorAddress)).wait()
        await (await proxy.confirmAggregator(aggregatorAddress)).wait()
        const revertedAggregator = await proxy.aggregator()
        expect(revertedAggregator).to.be.eq(aggregatorAddress)

        console.log(`Proxy Aggregator reverted from ${proposedAggregator} to ${revertedAggregator}`)
      } else {
        console.log('Wrong proxyRedirect method')
      }
    } else if (config.deploy) {
      // Deploy AggregatorProxy ////////////////////////////////////
      const deployConfig = config.deploy
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
          consumer,
          aggregatorProxyDeployment,
          name: deployConfig.name
        })
      }
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

export default func
func.id = 'deploy-aggregator'
func.tags = ['aggregator']
