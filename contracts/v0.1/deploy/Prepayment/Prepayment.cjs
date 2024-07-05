const path = require('node:path')
const {
  loadJson,
  loadMigration,
  updateMigration,
  validatePrepaymentDeployConfig,
} = require('../../scripts/utils.cjs')

const func = async function (hre) {
  const { deployments, getNamedAccounts, network } = hre
  const { deploy } = deployments
  const { deployer } = await getNamedAccounts()

  console.log('Prepayment.ts')

  const migrationDirPath = `./migration/${network.name}/Prepayment`
  const migrationFilesNames = await loadMigration(migrationDirPath)

  for (const migration of migrationFilesNames) {
    const config = await loadJson(path.join(migrationDirPath, migration))

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
        log: true,
      })
    }

    await updateMigration(migrationDirPath, migration)
  }
}

func.id = 'deploy-prepayment'
func.tags = ['prepayment']

module.exports = func
