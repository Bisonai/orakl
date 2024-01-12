const func = async function (hre) {
  const { deployments, getNamedAccounts } = hre;
  const { deploy } = deployments;
  const {
    deployer,
    vrfCoordinator: vrfCoordinatorAddress,
    requestResponseCoordinator: rrCoordinatorAddress,
    aggregatorRouter: aggregatorRouterAddress,
  } = await getNamedAccounts();

  console.log("0-VRFConsumer.ts");

  await deploy("VRFConsumer", {
    args: [
      aggregatorRouterAddress,
      rrCoordinatorAddress,
      vrfCoordinatorAddress,
    ],
    from: deployer,
    log: true,
  });
};

export default func;
func.id = "deploy-inspector-consumer";
func.tags = ["inspector-consumer"];
