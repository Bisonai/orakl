import { HardhatRuntimeEnvironment } from 'hardhat/types'
import { DeployFunction } from 'hardhat-deploy/types'

const func: DeployFunction = async function (hre: HardhatRuntimeEnvironment) {
  const { deployments, getNamedAccounts, network } = hre
  const { deploy } = deployments
  const { deployer } = await getNamedAccounts()

  console.log('1-RequestResponseCoordinator.ts')

  if (network.name == 'baobab') {
    console.log('Skipping')
    return
  }

  const requestResponseCoordinator = await deploy('RequestResponseCoordinator', {
    from: deployer,
    log: true
  })

  // TODO deploy only for tests
  const requestResponseConsumerMock = await deploy('RequestResponseConsumerMock', {
    args: [requestResponseCoordinator.address],
    from: deployer,
    log: true
  })
}

export default func
func.id = 'deploy-request-response'
func.tags = ['request-response']
