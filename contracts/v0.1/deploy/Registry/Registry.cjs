const func = async function (hre) {
  const { deployments, getNamedAccounts, network } = hre
  const { deploy } = deployments
  const { deployer } = await getNamedAccounts()

  const registryDeployment = await deploy('Registry', {
    args: [],
    from: deployer,
    log: true,
  })

  console.log('registeryDeployment:', registryDeployment)
}

func.id = 'deploy-registry'
func.tags = ['registry']

module.exports = func
