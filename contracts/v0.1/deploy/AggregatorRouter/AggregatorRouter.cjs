const path = require('node:path')
const {
  loadJson,
  loadMigration,
  updateMigration,
  loadDeployments,
} = require('../../scripts/utils.cjs')

const func = async function (hre) {
  console.log('AggregatorRouter')

  const { deployments, getNamedAccounts, network } = hre
  const { deploy } = deployments
  const { deployer } = await getNamedAccounts()

  const migrationDirPath = `./migration/${network.name}/AggregatorRouter`
  const migrationFilesNames = await loadMigration(migrationDirPath)

  for (const migration of migrationFilesNames) {
    const config = await loadJson(path.join(migrationDirPath, migration))
    let aggregatorRouter = undefined

    if (config.deploy) {
      console.log('deploy')

      const aggregatorRouterDeployment = await deploy('AggregatorRouter', {
        args: [],
        from: deployer,
        log: true,
      })

      aggregatorRouter = await ethers.getContractAt(
        'AggregatorRouter',
        aggregatorRouterDeployment.address,
      )

      console.log('AggregatorRouter:', aggregatorRouterDeployment)
    }

    aggregatorRouter = aggregatorRouter
      ? aggregatorRouter
      : await ethers.getContractAt('AggregatorRouter', config.aggregatorRouterAddress)

    if (config.updateProxies) {
      console.log('update proxies')
      const updateProxiesConfig = config.updateProxies

      const feedNames = []
      const addresses = []

      if (updateProxiesConfig.updateAll) {
        const deployments = await loadDeployments(`./deployments/${network.name}`)

        for (const key in deployments) {
          if (key.includes('AggregatorProxy')) {
            const feedName = key.split('_')[1]
            const address = deployments[key]
            feedNames.push(feedName)
            addresses.push(address)
          }
        }
      } else {
        for (const proxyObject of proxyList) {
          const feedName = proxyObject.feedName
          const address = proxyObject.name
          feedNames.push(feedName)
          addresses.push(address)
        }
      }
      if (feedNames.length == 0 || addresses.length == 0) {
        throw new Error('no proxies to update')
      }

      const tx = await (await aggregatorRouter.updateProxyBulk(feedNames, addresses)).wait()
      console.log(`bulk inserted feeds: ${tx.events[0].args[0]}`)
      console.log(`bulk inserted addresses:${tx.events[0].args[1]}`)
    }

    await updateMigration(migrationDirPath, migration)
  }
}

func.id = 'deploy-aggregator-router'
func.tags = ['aggregator-router']

module.exports = func
