import { HardhatRuntimeEnvironment } from 'hardhat/types'
import { DeployFunction } from 'hardhat-deploy/types'
import { loadJson } from '../scripts/v0.1/utils'
import { IRequestResponseConfig } from '../scripts/v0.1/types'

const func: DeployFunction = async function (hre: HardhatRuntimeEnvironment) {
  const { deployments, getNamedAccounts, network } = hre
  const { deploy } = deployments
  const { deployer, consumer } = await getNamedAccounts()

  console.log('1-RequestResponseCoordinator.ts')

  const requestResponseConfig: IRequestResponseConfig = await loadJson(
    `config/${network.name}/request-response.json`
  )

  if (network.name == 'baobab') {
    console.log('Skipping')
    return
  }

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

  // Configure Request-Resopnse coordinator
  console.log('Configure Request-Response coordinator')
  await (
    await requestResponseCoordinator.setConfig(
      requestResponseConfig.minimumRequestConfirmations,
      requestResponseConfig.maxGasLimit,
      requestResponseConfig.gasAfterPaymentCalculation,
      requestResponseConfig.feeConfig
    )
  ).wait()

  // TODO
  // Configure payment for direct Request-Response
  // await (await requestResponsecoordinator.setDirectPaymentConfig(vrfConfig.directPaymentConfig)).wait()

  // Add VRFCoordinator to Prepayment
  const prepaymentDeployerSigner = await ethers.getContractAt(
    'Prepayment',
    prepayment.address,
    deployer
  )
  await (await prepaymentDeployerSigner.addCoordinator(requestResponseCoordinator.address)).wait()

  if (['localhost', 'hardhat'].includes(network.name)) {
    await localhostDeployment({ deploy, requestResponseCoordinator, prepayment, consumer })
  }
}

async function localhostDeployment(args) {
  const { deploy, requestResponseCoordinator, prepayment, consumer } = args

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
