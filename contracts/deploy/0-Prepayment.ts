import { HardhatRuntimeEnvironment } from 'hardhat/types'
import { DeployFunction } from 'hardhat-deploy/types'

const func: DeployFunction = async function (hre: HardhatRuntimeEnvironment) {
  const { deployments, getNamedAccounts, network } = hre
  const { deploy } = deployments
  const { deployer } = await getNamedAccounts()

  console.log('0-Prepayment.ts')

  const prepaymentDeployment = await deploy('Prepayment', {
    from: deployer,
    log: true
  })
}

export default func
func.id = 'deploy-prepayment'
func.tags = ['prepayment']
