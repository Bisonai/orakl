const func = async function (hre) {
  const { deployments, getNamedAccounts, network } = hre
  const { deploy } = deployments
  const { deployer } = await getNamedAccounts()

  const registryDeployment = await deploy('L2Endpoint', {
    args: [],
    from: deployer,
    log: true
  })

  console.log('l2EndpointDeployment:', registryDeployment)
}

func.id = 'deploy-l2Endpoint'
func.tags = ['l2Endpoint']

module.exports = func
