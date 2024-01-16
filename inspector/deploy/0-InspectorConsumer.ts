import { HardhatRuntimeEnvironment } from "hardhat/types";
import { DeployFunction } from "hardhat-deploy/types";

const func: DeployFunction = async function (hre: HardhatRuntimeEnvironment) {
  const { deployments, getNamedAccounts } = hre;
  const { deploy } = deployments;
  const {
    deployer,
    vrfCoordinator: vrfCoordinatorAddress,
    rrCoordinator: rrCoordinatorAddress,
  } = await getNamedAccounts();

  console.log("0-InspectorConsumer.ts");

  await deploy("InspectorConsumer", {
    args: [rrCoordinatorAddress, vrfCoordinatorAddress],
    from: deployer,
    log: true,
  });
};

export default func;
func.id = "deploy-inspector-consumer";
func.tags = ["inspector-consumer"];
