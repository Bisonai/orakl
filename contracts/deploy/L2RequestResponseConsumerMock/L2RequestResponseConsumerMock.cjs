const path = require('node:path')
const { loadJson, loadMigration, updateMigration } = require('../../scripts/v0.1/utils.cjs')
const func = async function (hre) {
  const { deployments, getNamedAccounts, network } = hre
  const { deploy } = deployments
  const { deployer } = await getNamedAccounts()
  const migrationDirPath = `./migration/${network.name}/L2RequestResponseConsumerMock`
  const migrationFilesNames = await loadMigration(migrationDirPath)
  for (const migration of migrationFilesNames) {
    const config = await loadJson(path.join(migrationDirPath, migration))
    // Deploy L2 Consumer ////////////////////////////////////////////////////////
    if (config.deploy) {
      console.log('deploy')
      const deployConfig = config.deploy
      const l2VRFConsumerMock = await deploy('L2RequestResponseConsumerMock', {
        args: [deployConfig.l2EndpointAddress],
        from: deployer,
        log: true
      })

      console.log('L2RequestResponseConsumerMock:', l2VRFConsumerMock)
    }

    await updateMigration(migrationDirPath, migration)
  }
}

func.id = 'deploy-consumer'
func.tags = ['consumer']

module.exports = func
