const path = require('node:path')
const {
  loadJson,
  loadMigration,
  updateMigration,
  validateCoordinatorDeployConfig,
  validateSetConfig,
} = require('../../scripts/utils.cjs')

const func = async function (hre) {
  const { deployments, getNamedAccounts, network } = hre
  const { deploy } = deployments
  const { deployer } = await getNamedAccounts()

  console.log('RequestResponseCoordinator.ts')

  const migrationDirPath = `./migration/${network.name}/RequestResponse`
  const migrationFilesNames = await loadMigration(migrationDirPath)

  for (const migration of migrationFilesNames) {
    const config = await loadJson(path.join(migrationDirPath, migration))

    const prepayment = await ethers.getContract('Prepayment')
    let requestResponseCoordinator = undefined

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
        log: true,
      })

      requestResponseCoordinator = await ethers.getContractAt(
        'RequestResponseCoordinator',
        requestResponseDeployment.address,
      )

      // RequestResponseConsumerMock
      if (['localhost', 'hardhat'].includes(network.name)) {
        await localhostDeployment({
          deploy,
          requestResponseCoordinator,
          prepayment,
        })
      }
    }

    requestResponseCoordinator = requestResponseCoordinator
      ? requestResponseCoordinator
      : await ethers.getContractAt(
          'RequestResponseCoordinator',
          config.requestResponseCoordinatorAddress,
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
          setConfig.feeConfig,
        )
      ).wait()
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
        deployer,
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
  const { deploy, requestResponseCoordinator, prepayment } = args

  const requestResponseConsumerMockDeployment = await deploy('RequestResponseConsumerMock', {
    contract: 'RequestResponseConsumerMock',
    args: [requestResponseCoordinator.address],
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
    await prepaymentConsumerSigner.addConsumer(accId, requestResponseConsumerMockDeployment.address)
  ).wait()
}

func.id = 'deploy-request-response'
func.tags = ['request-response']

module.exports = func
