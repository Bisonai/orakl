import { HardhatRuntimeEnvironment } from 'hardhat/types'
import { DeployFunction } from 'hardhat-deploy/types'
import { expect } from 'chai'
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

    // Deploy Aggregator ////////////////////////////////////
    if (config.deploy && (!config.redirectProxy || config.redirectProxy.status == 'initial')) {
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
      const proxyFile = redirectProxyConfig.proxyFileName
      const proxy = await ethers.getContract(proxyFile)
      const oldAggregatorAddress = redirectProxyConfig.oldAggregatorAddress
      const newAggregatorAddress = aggregator
        ? aggregator.address
        : redirectProxyConfig.newAggregatorAddress

      if (redirectProxyConfig.status == 'initial') {
        // Propose new aggregator
        expect(await proxy.aggregator()).to.be.eq(config.redirectProxy.oldAggregatorAddress)
        await (await proxy.proposeAggregator(newAggregatorAddress)).wait()

        const proposedAggregator = await proxy.proposedAggregator()
        expect(proposedAggregator).to.be.eq(newAggregatorAddress)

        console.log(`Proposed proxy aggregator:${proposedAggregator}`)
      } else if (redirectProxyConfig.status == 'confirm') {
        // Confirm new aggregator from Proxy
        expect(await proxy.aggregator()).to.be.eq(oldAggregatorAddress)
        expect(await proxy.proposedAggregator()).to.be.eq(newAggregatorAddress)
        await (await proxy.confirmAggregator(newAggregatorAddress)).wait()

        const confirmedAggregator = await proxy.aggregator()
        expect(confirmedAggregator).to.be.eq(newAggregatorAddress)

        console.log(
          `Proxy Aggregator redirected from ${oldAggregatorAddress} to new ${confirmedAggregator}`
        )
      } else if (redirectProxyConfig.status == 'revert') {
        // Revert back to old Aggregator Address
        expect(await proxy.aggregator()).to.be.eq(config.redirectProxy.newAggregatorAddress)
        await (await proxy.proposeAggregator(oldAggregatorAddress)).wait()
        await (await proxy.confirmAggregator(oldAggregatorAddress)).wait()
        const revertedAggregator = await proxy.aggregator()
        expect(revertedAggregator).to.be.eq(oldAggregatorAddress)

        console.log(
          `Proxy Aggregator reverted from ${newAggregatorAddress} to ${revertedAggregator}`
        )
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
