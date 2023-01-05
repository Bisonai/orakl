import { HardhatRuntimeEnvironment } from 'hardhat/types'
import { DeployFunction } from 'hardhat-deploy/types'
const vrfConfig = require('../config/vrf.json')

const func: DeployFunction = async function (hre: HardhatRuntimeEnvironment) {
  const { deployments, getNamedAccounts } = hre
  const { deploy } = deployments
  const { deployer, consumer } = await getNamedAccounts()

  console.log('2-VRFCoordinator.ts')

  const vrfCoordinatorDeployment = await deploy('VRFCoordinator', {
    from: deployer,
    log: true
  })

  const vrfCoordinator = await ethers.getContractAt(
    'VRFCoordinator',
    vrfCoordinatorDeployment.address
  )

  // Register proving key
  console.log('Register proving key')
  for (const oracle of vrfConfig.oracle) {
    await vrfCoordinator.registerProvingKey(oracle.address, oracle.publicProvingKey)
  }

  // Configure VRF coordinator
  console.log('Configure VRF coordinator')
  await vrfCoordinator.setConfig(
    vrfConfig.minimumRequestConfirmations,
    vrfConfig.maxGasLimit,
    vrfConfig.gasAfterPaymentCalculation,
    vrfConfig.feeConfig
  )

  // TODO deploy only for tests
  const vrfConsumerMockDeployment = await deploy('VRFConsumerMock', {
    args: [vrfCoordinator.address],
    from: consumer,
    log: true
  })

  const vrfCoordinatorConsumerSigner = await ethers.getContractAt(
    'VRFCoordinator',
    vrfCoordinatorDeployment.address,
    consumer
  )

  // Create subscription
  const subscriptionReceipt = await (await vrfCoordinatorConsumerSigner.createSubscription()).wait()
  const { subId } = subscriptionReceipt.events[0].args

  // Add consumer to subscription
  await vrfCoordinatorConsumerSigner.addConsumer(subId, vrfConsumerMockDeployment.address)
}

export default func
func.id = 'deploy-vrf'
func.tags = ['vrf']
