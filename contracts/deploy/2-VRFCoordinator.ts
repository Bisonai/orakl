import { HardhatRuntimeEnvironment } from 'hardhat/types'
import { DeployFunction } from 'hardhat-deploy/types'
import { loadJson } from '../scripts/v0.1/utils'
import { IVrfConfig } from '../scripts/v0.1/types'

async function localhostDeployment(args) {
  const { deploy, vrfCoordinator, prepayment, consumer } = args
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

const func: DeployFunction = async function (hre: HardhatRuntimeEnvironment) {
  const { deployments, getNamedAccounts, network } = hre
  const { deploy } = deployments
  const { deployer, consumer } = await getNamedAccounts()

  console.log('2-VRFCoordinator.ts')

  const vrfConfig: IVrfConfig = await loadJson(`config/${network.name}/vrf.json`)

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
    console.log('keyHash', tx.events[0].args.keyHash)
    console.log('oracle', tx.events[0].args.oracle)
  }

  // Configure VRF coordinator
  console.log('Configure VRF coordinator')
  await (
    await vrfCoordinator.setConfig(
      vrfConfig.minimumRequestConfirmations,
      vrfConfig.maxGasLimit,
      vrfConfig.gasAfterPaymentCalculation,
      vrfConfig.feeConfig
    )
  ).wait()

  // Configure payment for direct VRF request
  await (await vrfCoordinator.setDirectPaymentConfig(vrfConfig.directPaymentConfig)).wait()

  // Add VRFCoordinator to Prepayment
  const prepaymentDeployerSigner = await ethers.getContractAt(
    'Prepayment',
    prepayment.address,
    deployer
  )
  await (await prepaymentDeployerSigner.addCoordinator(vrfCoordinatorDeployment.address)).wait()

  if (['localhost', 'hardhat'].includes(network.name)) {
    await localhostDeployment({ deploy, vrfCoordinator, consumer, prepayment })
  }
}

export default func
func.id = 'deploy-vrf'
func.tags = ['vrf']
