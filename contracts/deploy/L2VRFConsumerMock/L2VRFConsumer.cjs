const func = async function (hre) {
  const { deployments, getNamedAccounts, network } = hre
  const { deploy } = deployments
  const { deployer } = await getNamedAccounts()

  const registryDeployment = await deploy('L2VRFConsumerMock', {
    args: ['0xccD917Bb5312d42260CD77c8DFc105293a37F9B5'],
    from: deployer,
    log: true
  })

  console.log('l2VrfConsumerDeployment:', registryDeployment)
}

func.id = 'deploy-consumer'
func.tags = ['consumer']

module.exports = func
