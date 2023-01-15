import { HardhatRuntimeEnvironment } from 'hardhat/types'
import { DeployFunction } from 'hardhat-deploy/types'
const vrfConfig = require('../config/vrf.json')

const func: DeployFunction = async function (hre: HardhatRuntimeEnvironment) {
  const { deployments, getNamedAccounts, network } = hre
  const { deploy } = deployments
  const { deployer, consumer } = await getNamedAccounts()

  console.log('2-VRFCoordinator.ts')

  if (network.name == 'baobab') {
    console.log('Skipping')
    return
  }

  const prepayment = await ethers.getContract('Prepayment')

  const vrfCoordinatorDeployment = await deploy('VRFCoordinator', {
    args: [prepayment.address],
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
    const tx = await (
      await vrfCoordinator.registerProvingKey(oracle.address, oracle.publicProvingKey)
    ).wait()
    console.log(tx)
    console.log(tx.events[0].args)
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

  const prepaymentConsumerSigner = await ethers.getContractAt(
    'Prepayment',
    prepayment.address,
    consumer
  )

  // Create account
  const accountReceipt = await (await prepaymentConsumerSigner.createAccount()).wait()
  const { accId } = accountReceipt.events[0].args

  // Deposit 1 KLAY
  await prepaymentConsumerSigner.deposit(accId, { value: ethers.utils.parseUnits('1', 'ether') })

  // Add consumer to account
  await prepaymentConsumerSigner.addConsumer(accId, vrfConsumerMockDeployment.address)

  // Add VRFCoordinator to Prepayment
  const prepaymentDeployerSigner = await ethers.getContractAt(
    'Prepayment',
    prepayment.address,
    deployer
  )

  await prepaymentDeployerSigner.addCoordinator(vrfCoordinatorDeployment.address)
}

export default func
func.id = 'deploy-vrf'
func.tags = ['vrf']
