const path = require('node:path')
const {
  loadJson,
  loadMigration,
  updateMigration,
  validateCoordinatorDeployConfig,
  validateSetConfig,
  validateVrfDeregisterOracle,
  validateVrfRegisterOracle,
} = require('../../scripts/utils.cjs')

const func = async function (hre) {
  const { deployments, getNamedAccounts, network } = hre
  const { deploy } = deployments
  const { deployer } = await getNamedAccounts()

  console.log('VRFCoordinator.ts')

  const migrationDirPath = `./migration/${network.name}/VRF`
  const migrationFilesNames = await loadMigration(migrationDirPath)

  for (const migration of migrationFilesNames) {
    const config = await loadJson(path.join(migrationDirPath, migration))

    const prepayment = await ethers.getContract('Prepayment')
    let vrfCoordinator = undefined

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
        log: true,
      })

      vrfCoordinator = await ethers.getContractAt(
        'VRFCoordinator',
        vrfCoordinatorDeployment.address,
      )

      // VRFConsumermock
      if (['localhost', 'hardhat'].includes(network.name)) {
        await localhostDeployment({
          deploy,
          vrfCoordinator,
          prepayment,
          name: deployConfig.version,
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
          `Oracle registered with address=${tx.events[0].args.oracle} and keyHash=${tx.events[0].args.keyHash}`,
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
          `Oracle deregistered with address=${tx.events[0].args.oracle} and keyHash=${tx.events[0].args.keyHash}`,
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
          setConfig.feeConfig,
        )
      ).wait()
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
        deployer,
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
    log: true,
  })

  const prepaymentConsumerSigner = await ethers.getContractAt(
    'Prepayment',
    prepayment.address,
    consumer,
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

func.id = 'deploy-vrf'
func.tags = ['vrf']

module.exports = func
