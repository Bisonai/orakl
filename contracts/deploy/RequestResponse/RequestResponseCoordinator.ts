import { HardhatRuntimeEnvironment } from 'hardhat/types'
import { DeployFunction } from 'hardhat-deploy/types'
import * as path from 'node:path'
import {
  loadJson,
  loadMigration,
  updateMigration,
  validateCoordinatorDeployConfig,
  validateDirectPaymentConfig,
  validateMinBalanceConfig,
  validateSetConfig
} from '../../scripts/v0.1/utils'
import { IRequestResponseCoordinatorConfig } from '../../scripts/v0.1/types'

const func: DeployFunction = async function (hre: HardhatRuntimeEnvironment) {
  const { deployments, getNamedAccounts, network } = hre
  const { deploy } = deployments
  const { deployer, consumer } = await getNamedAccounts()

  console.log('RequestResponseCoordinator.ts')

  const migrationDirPath = `./migration/${network.name}/RequestResponse`
  const migrationFilesNames = await loadMigration(migrationDirPath)

  for (const migration of migrationFilesNames) {
    const config: IRequestResponseCoordinatorConfig = await loadJson(
      path.join(migrationDirPath, migration)
    )

    const prepayment = await ethers.getContract('Prepayment')
    let requestResponseCoordinator: ethers.Contract = undefined

    // Deploy RequestResponseCoordinator ////////////////////////////////////////
    if (config.deploy) {
      console.log('deploy')
      const deployConfig = config.deploy
      if (!validateCoordinatorDeployConfig(deployConfig)) {
        throw new Error('Invalid RRC deploy config')
      }

      const requestResponseCoordinatorName = `RequestResponseCoordinator_${deployConfig.version}`

      const requestResponseDeployment = await deploy(requestResponseCoordinatorName, {
        contract: 'RequestResponseCoordinator',
        args: [prepayment.address],
        from: deployer,
        log: true
      })

      requestResponseCoordinator = await ethers.getContractAt(
        'RequestResponseCoordinator',
        requestResponseDeployment.address
      )

      // RequestResponseConsumerMock
      if (['localhost', 'hardhat'].includes(network.name)) {
        await localhostDeployment({
          deploy,
          requestResponseCoordinator,
          prepayment,
          name: deployConfig.version
        })
      }
    }

    requestResponseCoordinator = requestResponseCoordinator
      ? requestResponseCoordinator
      : await ethers.getContractAt(
          'RequestResponseCoordinator',
          config.requestResponseCoordinatorAddress
        )

    // Register Oracle //////////////////////////////////////////////////////////
    if (config.registerOracle) {
      console.log('registerOracle')

      for (const oracle of config.registerOracle) {
        const tx = await (await requestResponseCoordinator.registerOracle(oracle)).wait()
        console.log('Oracle Registered', tx.events[0].args.oracle)
      }
    }

    // Deregister Oracle ////////////////////////////////////////////////////////
    if (config.deregisterOracle) {
      console.log('deregisterOracle')

      for (const oracle of config.deregisterOracle) {
        const tx = await (await requestResponseCoordinator.deregisterOracle(oracle)).wait()
        console.log('Oracle Deregistered', tx.events[0].args.oracle)
      }
    }

    // Configure Request-Response coordinator ///////////////////////////////////
    if (config.setConfig) {
      console.log('setConfig')
      const setConfig = config.setConfig
      if (!validateSetConfig(setConfig)) {
        throw new Error('Invalid RRC setConfig config')
      }

      await (
        await requestResponseCoordinator.setConfig(
          setConfig.maxGasLimit,
          setConfig.gasAfterPaymentCalculation,
          setConfig.feeConfig
        )
      ).wait()
    }

    // setDirectPaymentConfig ///////////////////////////////////////////////////
    if (config.setDirectPaymentConfig) {
      console.log('setDirectPaymentConfig')
      const setDirectPaymentConfig = config.setDirectPaymentConfig
      if (!validateDirectPaymentConfig(setDirectPaymentConfig)) {
        throw new Error('Invalid RRC setDirectPaymentConfig config')
      }

      await (
        await requestResponseCoordinator.setDirectPaymentConfig(
          setDirectPaymentConfig.directPaymentConfig
        )
      ).wait()
    }

    // setMinBalance ////////////////////////////////////////////////////////////
    if (config.setMinBalance) {
      console.log('setMinBalance')
      const setMinBalanceConfig = config.setMinBalance
      if (!validateMinBalanceConfig(setMinBalanceConfig)) {
        throw new Error('Invalid RRC setMinBalance config')
      }

      await (await requestResponseCoordinator.setMinBalance(setMinBalanceConfig.minBalance)).wait()
    }

    // Add RequestResponseCoordinator to Prepayment /////////////////////////////
    if (config.addCoordinator) {
      console.log('addCoordinator')
      const addCoordinatorConfig = config.addCoordinator

      const requestResponseCoordinatorAddress = requestResponseCoordinator
        ? requestResponseCoordinator.address
        : addCoordinatorConfig.coordinatorAddress
      if (!requestResponseCoordinatorAddress) {
        throw new Error('requestResponseCoordinator address is undefined')
      }

      const prepaymentDeployerSigner = await ethers.getContractAt(
        'Prepayment',
        prepayment.address,
        deployer
      )
      await (
        await prepaymentDeployerSigner.addCoordinator(requestResponseCoordinatorAddress)
      ).wait()
    }

    await updateMigration(migrationDirPath, migration)
  }
}

async function localhostDeployment(args) {
  const { consumer } = await getNamedAccounts()
  const { deploy, requestResponseCoordinator, prepayment, name } = args
  const requestResponseConsumerMockName = `RequestResponseConsumerMock_${name}`

  const requestResponseConsumerMockDeployment = await deploy(requestResponseConsumerMockName, {
    contract: 'RequestResponseConsumerMock',
    args: [requestResponseCoordinator.address],
    from: consumer,
    log: true
  })

  const prepaymentConsumerSigner = await ethers.getContractAt(
    'Prepayment',
    prepayment.address,
    consumer
  )

  // Create account
  const accountReceipt = await (await prepaymentConsumerSigner.createAccount()).wait()
  const { accId } = accountReceipt.events[0].args

  // Deposit 1 KLAY
  await (
    await prepaymentConsumerSigner.deposit(accId, { value: ethers.utils.parseUnits('1', 'ether') })
  ).wait()

  // Add consumer to account
  await (
    await prepaymentConsumerSigner.addConsumer(accId, requestResponseConsumerMockDeployment.address)
  ).wait()
}

export default func
func.id = 'deploy-request-response'
func.tags = ['request-response']
