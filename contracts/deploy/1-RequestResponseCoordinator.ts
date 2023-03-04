import { HardhatRuntimeEnvironment } from 'hardhat/types'
import { DeployFunction } from 'hardhat-deploy/types'
import { loadJson } from '../scripts/v0.1/utils'
import { IRequestResponseConfig } from '../scripts/v0.1/types'

const func: DeployFunction = async function (hre: HardhatRuntimeEnvironment) {
  const { deployments, getNamedAccounts, network } = hre
  const { deploy } = deployments
  const { deployer, consumer } = await getNamedAccounts()

  console.log('1-RequestResponseCoordinator.ts')

  const config: IRequestResponseConfig = await loadJson(
    `config/${network.name}/request-response.json`
  )

  const prepayment = await ethers.getContract('Prepayment')

  const requestResponseDeployment = await deploy('RequestResponseCoordinator', {
    args: [prepayment.address],
    from: deployer,
    log: true
  })

  const requestResponseCoordinator = await ethers.getContractAt(
    'RequestResponseCoordinator',
    requestResponseDeployment.address
  )

  // Register oracle
  console.log('Register oracle')
  for (const oracle of config.oracle) {
    const tx = await (await requestResponseCoordinator.registerOracle(oracle.address)).wait()
    console.log('oracle', tx.events[0].args.oracle)
  }

  // Configure Request-Response coordinator
  console.log('Configure Request-Response coordinator')
  await (
    await requestResponseCoordinator.setConfig(
      config.maxGasLimit,
      config.gasAfterPaymentCalculation,
      config.feeConfig
    )
  ).wait()

  // Configure payment for direct Request-Response
  await (await requestResponseCoordinator.setDirectPaymentConfig(config.directPaymentConfig)).wait()
  // Configure minBalance
  await (await requestResponseCoordinator.setMinBalance(config.minBalance)).wait()
  // Add RequestResponseCoordinator to Prepayment
  const prepaymentDeployerSigner = await ethers.getContractAt(
    'Prepayment',
    prepayment.address,
    deployer
  )
  await (await prepaymentDeployerSigner.addCoordinator(requestResponseCoordinator.address)).wait()

  // Localhost deployment
  if (['localhost', 'hardhat'].includes(network.name)) {
    await localhostDeployment({ deploy, requestResponseCoordinator, prepayment })
  }
}

async function localhostDeployment(args) {
  const { consumer } = await getNamedAccounts()
  const { deploy, requestResponseCoordinator, prepayment } = args

  const requestResponseConsumerMockDeployment = await deploy('RequestResponseConsumerMock', {
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
