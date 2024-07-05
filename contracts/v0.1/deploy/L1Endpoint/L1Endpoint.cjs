const path = require('node:path')
const { loadJson, loadMigration, updateMigration } = require('../../scripts/utils.cjs')

const func = async function (hre) {
  const { deployments, getNamedAccounts, network } = hre
  const { deploy } = deployments
  const { deployer } = await getNamedAccounts()
  const migrationDirPath = `./migration/${network.name}/L1Endpoint`
  const migrationFilesNames = await loadMigration(migrationDirPath)
  for (const migration of migrationFilesNames) {
    const config = await loadJson(path.join(migrationDirPath, migration))
    // Deploy L1Endpoint ////////////////////////////////////////////////////////
    if (config.deploy) {
      console.log('deploy')
      const deployConfig = config.deploy
      const l1Endpoint = await deploy('L1Endpoint', {
        args: [
          deployConfig.registry,
          deployConfig.vrfCoordinator,
          deployConfig.requestResponseCoordinator,
        ],
        from: deployer,
        log: true,
      })

      console.log('L1Endpoint:', l1Endpoint)
    }

    await updateMigration(migrationDirPath, migration)
  }
}

func.id = 'deploy-L1Endpoint'
func.tags = ['L1Endpoint']

module.exports = func
