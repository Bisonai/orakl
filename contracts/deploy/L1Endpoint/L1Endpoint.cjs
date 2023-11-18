const path = require('node:path')
const { loadJson, loadMigration, updateMigration } = require('../../scripts/v0.1/utils.cjs')

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
      const l1EndpointDeployment = await deploy('L1Endpoint', {
        args: [
          '0xDA8c0A00A372503aa6EC80f9b29Cc97C454bE499',
          '0x89c589256AcaC342c641Cd472Fd8d07550d347a8'
        ],
        from: deployer,
        log: true
      })

      console.log('l1EndpointDeployment:', l1EndpointDeployment)
    }

    await updateMigration(migrationDirPath, migration)
  }
}

func.id = 'deploy-l1Endpoint'
func.tags = ['l1Endpoint']

module.exports = func
