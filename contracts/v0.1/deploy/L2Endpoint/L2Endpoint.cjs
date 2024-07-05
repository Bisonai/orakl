const path = require('node:path')
const { loadJson, loadMigration, updateMigration } = require('../../scripts/utils.cjs')

const func = async function (hre) {
  const { deployments, getNamedAccounts, network } = hre
  const { deploy } = deployments
  const { deployer } = await getNamedAccounts()
  const migrationDirPath = `./migration/${network.name}/L2Endpoint`
  const migrationFilesNames = await loadMigration(migrationDirPath)
  for (const migration of migrationFilesNames) {
    const config = await loadJson(path.join(migrationDirPath, migration))
    // Deploy L2Endpoint ////////////////////////////////////////////////////////
    if (config.deploy) {
      console.log('deploy')
      const l2EndpointDeployment = await deploy('L2Endpoint', {
        args: [],
        from: deployer,
        log: true,
      })

      console.log('l2EndpointDeployment:', l2EndpointDeployment)
    }

    await updateMigration(migrationDirPath, migration)
  }
}

func.id = 'deploy-l2Endpoint'
func.tags = ['l2Endpoint']

module.exports = func
