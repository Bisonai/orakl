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
  validateSetConfig,
  validateVrfDeregisterOracle,
  validateVrfRegisterOracle
} from '../../scripts/v0.1/utils'
import { IVrfCoordinatorConfig } from '../../scripts/v0.1/types'

const func: DeployFunction = async function (hre: HardhatRuntimeEnvironment) {
  const { deployments, getNamedAccounts, network } = hre
  const { deploy } = deployments
  const { deployer } = await getNamedAccounts()

  console.log('VRFCoordinator.ts')

  const migrationDirPath = `./migration/${network.name}/VRF`
  const migrationFilesNames = await loadMigration(migrationDirPath)

  for (const migration of migrationFilesNames) {
    const config: IVrfCoordinatorConfig = await loadJson(path.join(migrationDirPath, migration))

    const prepayment = await ethers.getContract('Prepayment')
    let vrfCoordinator: ethers.Contract = undefined

    // Deploy VRFCoordinator ////////////////////////////////////////////////////
    if (config.deploy) {
      console.log('deploy')
      const deployConfig = config.deploy
      if (!validateCoordinatorDeployConfig(deployConfig)) {
        throw new Error('Invalid VRF deploy config')
      }

      const vrfCoordinatorName = `VRFCoordinator_${deployConfig.version}`

      const vrfCoordinatorDeployment = await deploy(vrfCoordinatorName, {
        contract: 'VRFCoordinator',
        args: [prepayment.address],
        from: deployer,
        log: true
      })

      vrfCoordinator = await ethers.getContractAt(
        'VRFCoordinator',
        vrfCoordinatorDeployment.address
      )

      // VRFConsumermock
      if (['localhost', 'hardhat'].includes(network.name)) {
        await localhostDeployment({
          deploy,
          vrfCoordinator,
          prepayment,
          name: deployConfig.version
        })
      }
    }

    vrfCoordinator = vrfCoordinator
      ? vrfCoordinator
      : await ethers.getContractAt('VRFCoordinator', config.vrfCoordinatorAddress)

    // Register oracle //////////////////////////////////////////////////////////
    if (config.registerOracle) {
      console.log('registerOracle')
      const registerOracleConfig = config.registerOracle
      if (!validateVrfRegisterOracle(registerOracleConfig)) {
        throw new Error('Invalid VRF registerOracle config')
      }

      for (const oracle of registerOracleConfig) {
        const tx = await (
          await vrfCoordinator.registerOracle(oracle.address, oracle.publicProvingKey)
        ).wait()
        console.log(
          `Oracle registered with address=${tx.events[0].args.oracle} and keyHash=${tx.events[0].args.keyHash}`
        )
      }
    }

    // Deregister oracle ////////////////////////////////////////////////////////
    if (config.deregisterOracle) {
      console.log('deregisterOracle')
      const deregisterOracleConfig = config.deregisterOracle
      if (!validateVrfDeregisterOracle(deregisterOracleConfig)) {
        throw new Error('Invalid VRF deregisterOracle config')
      }

      for (const oracle of deregisterOracleConfig) {
        const tx = await (await vrfCoordinator.deregisterOracle(oracle.address)).wait()
        console.log(
          `Oracle deregistered with address=${tx.events[0].args.oracle} and keyHash=${tx.events[0].args.keyHash}`
        )
      }
    }

    // Configure VRF coordinator ////////////////////////////////////////////////
    if (config.setConfig) {
      console.log('setConfig')
      const setConfig = config.setConfig
      if (!validateSetConfig(setConfig)) {
        throw new Error('Invalid VRF setConfig config')
      }

      await (
        await vrfCoordinator.setConfig(
          setConfig.maxGasLimit,
          setConfig.gasAfterPaymentCalculation,
          setConfig.feeConfig
        )
      ).wait()
    }

    // Configure payment for direct VRF request /////////////////////////////////
    if (config.setDirectPaymentConfig) {
      console.log('setDirectPaymentConfig')
      const setDirectPaymentConfig = config.setDirectPaymentConfig
      if (!validateDirectPaymentConfig(setDirectPaymentConfig)) {
        throw new Error('Invalid VRF setDirectPaymentConfig config')
      }

      await (
        await vrfCoordinator.setDirectPaymentConfig(setDirectPaymentConfig.directPaymentConfig)
      ).wait()
    }

    // Configure minBalance
    if (config.setMinBalance) {
      console.log('setMinBalance')
      const setMinBalanceConfig = config.setMinBalance
      if (!validateMinBalanceConfig(setMinBalanceConfig)) {
        throw new Error('Invalid RRC setMinBalance config')
      }

      await (await vrfCoordinator.setMinBalance(setMinBalanceConfig.minBalance)).wait()
    }

    // Add VRFCoordinator to Prepayment /////////////////////////////////////////
    if (config.addCoordinator) {
      console.log('addCoordinator')
      const addCoordinatorConfig = config.addCoordinator

      const vrfCoordinatorAddress = vrfCoordinator
        ? vrfCoordinator.address
        : addCoordinatorConfig.coordinatorAddress
      if (!vrfCoordinatorAddress) {
        throw new Error('VRFCoordinator address is undefined')
      }

      const prepaymentDeployerSigner = await ethers.getContractAt(
        'Prepayment',
        prepayment.address,
        deployer
      )
      await (await prepaymentDeployerSigner.addCoordinator(vrfCoordinatorAddress)).wait()
    }

    await updateMigration(migrationDirPath, migration)
  }
}

async function localhostDeployment(args) {
  const { consumer } = await getNamedAccounts()
  const { deploy, vrfCoordinator, prepayment } = args
  const vrfConsumerMockDeployment = await deploy('VRFConsumerMock', {
    args: [vrfCoordinator.address],
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
    await prepaymentConsumerSigner.addConsumer(accId, vrfConsumerMockDeployment.address)
  ).wait()
}

export default func
func.id = 'deploy-vrf'
func.tags = ['vrf']
