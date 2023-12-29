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

        for (const key in deployments) {
          if (key.includes('AggregatorProxy')) {
            const feedName = key.split('_')[1]
            const address = deployments[key]
            const tx = await (await dataFeedRouter.updateProxy(feedName, address)).wait()
            console.log(
              `Proxy Registered {feedName: ${tx.events[0].args.feedName}, address: ${tx.events[0].args.proxyAddress}}`
            )
          }
        }
      } else {
        if (!updateProxiesConfig.proxyList) {
          throw new Error('proxy list is empty')
        }
        for (const proxyObject of proxyList) {
          const feedName = proxyObject.feedName
          const address = proxyObject.name
          const tx = await (await dataFeedRouter.updateProxy(feedName, address)).wait()
          console.log(
            `Proxy Registered {feedName: ${tx.events[0].args.feedName}, address: ${tx.events[0].args.proxyAddress}}`
          )
        }
      }
    }

    await updateMigration(migrationDirPath, migration)
  }
}

func.id = 'deploy-data-feed-router'
func.tags = ['data-feed-router']

module.exports = func
