const path = require('node:path')
const {
  loadJson,
  loadMigration,
  updateMigration,
  loadDeployments
} = require('../../scripts/v0.1/utils.cjs')

const func = async function (hre) {
  console.log('DataFeedRouter')

  const { deployments, getNamedAccounts, network } = hre
  const { deploy } = deployments
  const { deployer } = await getNamedAccounts()

  const migrationDirPath = `./migration/${network.name}/DataFeedRouter`
  const migrationFilesNames = await loadMigration(migrationDirPath)

  for (const migration of migrationFilesNames) {
    const config = await loadJson(path.join(migrationDirPath, migration))
    let dataFeedRouter = undefined

    if (config.deploy) {
      console.log('deploy')

      const dataFeedRouterDeployment = await deploy('DataFeedRouter', {
        args: [],
        from: deployer,
        log: true
      })

      dataFeedRouter = await ethers.getContractAt(
        'DataFeedRouter',
        dataFeedRouterDeployment.address
      )

      console.log('DataFeedRouter:', dataFeedRouterDeployment)
    }

    dataFeedRouter = dataFeedRouter
      ? dataFeedRouter
      : await ethers.getContractAt('DataFeedRouter', config.dataFeedRouterAddress)

    if (config.updateProxies) {
      console.log('update proxies')
      const updateProxiesConfig = config.updateProxies

      if (updateProxiesConfig.updateAll) {
        const deployments = await loadDeployments(`./deployments/${network.name}`)

        const feedNames = []
        const addresses = []
        for (const key in deployments) {
          if (key.includes('AggregatorProxy')) {
            const feedName = key.split('_')[1]
            const address = deployments[key]
            feedNames.push(feedName)
            addresses.push(address)
          }
        }
        const tx = await (await dataFeedRouter.updateProxyBulk(feedNames, addresses)).wait()
        console.log(`bulk inserted feeds: ${tx.events[0].args[0]}`)
        console.log(`bulk inserted addresses:${tx.events[0].args[1]}`)
      } else {
        if (!updateProxiesConfig.proxyList) {
          throw new Error('proxy list is empty')
        }

        const feedNames = []
        const addresses = []
        for (const proxyObject of proxyList) {
          const feedName = proxyObject.feedName
          const address = proxyObject.name
          feedNames.push(feedName)
          addresses.push(address)
        }
        const tx = await (await dataFeedRouter.updateProxyBulk(feedNames, addresses)).wait()
        console.log(`bulk inserted feeds: ${tx.events[0].args[0]}`)
        console.log(`bulk inserted addresses:${tx.events[0].args[1]}`)
      }
    }

    await updateMigration(migrationDirPath, migration)
  }
}

func.id = 'deploy-data-feed-router'
func.tags = ['data-feed-router']

module.exports = func
