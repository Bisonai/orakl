import { HardhatRuntimeEnvironment } from 'hardhat/types'
import { DeployFunction } from 'hardhat-deploy/types'
import * as path from 'node:path'
import {
  loadJson,
  loadMigration,
  updateMigration,
  validatePrepaymentDeployConfig
} from '../../scripts/v0.1/utils'
import { IPrepaymentConfig } from '../../scripts/v0.1/types'

const func: DeployFunction = async function (hre: HardhatRuntimeEnvironment) {
  const { deployments, getNamedAccounts, network } = hre
  const { deploy } = deployments
  const { deployer } = await getNamedAccounts()

  console.log('Prepayment.ts')

  const migrationDirPath = `./migration/${network.name}/Prepayment`
  const migrationFilesNames = await loadMigration(migrationDirPath)

  for (const migration of migrationFilesNames) {
    const config: IPrepaymentConfig = await loadJson(path.join(migrationDirPath, migration))

    // Deploy Prepayment ////////////////////////////////////////////////////////
    if (config.deploy) {
      console.log('deploy')
      const deployConfig = config.deploy

      if (!validatePrepaymentDeployConfig(deployConfig)) {
        throw new Error('Invalid Prepayment deploy config')
      }

      const prepaymentDeployment = await deploy('Prepayment', {
        args: [deployConfig.protocolFeeRecipient],
        from: deployer,
        log: true
      })
    }

    await updateMigration(migrationDirPath, migration)
  }
}

export default func
func.id = 'deploy-prepayment'
func.tags = ['prepayment']
