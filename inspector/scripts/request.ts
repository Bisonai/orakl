import { deployments, ethers, getNamedAccounts } from "hardhat";
import {
  getKeyHash,
  estimateVRFServiceFee,
  estimateRRServiceFee,
} from "./utils";

async function main() {
  //await deployments.fixture("InspectorConsumer");

  const _inspectorConsumer = await deployments.get("InspectorConsumer");
  const inspectorConsumer = await ethers.getContractAt(
    _inspectorConsumer.abi,
    _inspectorConsumer.address
  );

  const keyHash = getKeyHash();
  const callbackGasLimit = 500_000;
  const numWords = 1;

  const { deployer } = await getNamedAccounts();
  const estimatedVRFServiceFee = await estimateVRFServiceFee();

  let txReceipt = await (
    await inspectorConsumer.requestVRFDirect(
      keyHash,
      callbackGasLimit,
      numWords,
      deployer,
      {
        value: ethers.parseEther(estimatedVRFServiceFee.toString()),
      }
    )
  ).wait();

  console.log(txReceipt);
  console.log("Requested random words using direct payment");

  const estimatedRRServiceFee = await estimateRRServiceFee();
  txReceipt = await (
    await inspectorConsumer.requestRRDirect(callbackGasLimit, {
      value: ethers.parseEther(estimatedRRServiceFee.toString()),
    })
  ).wait();

  console.log(txReceipt);
  console.log("Requested data using direct payment");
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
