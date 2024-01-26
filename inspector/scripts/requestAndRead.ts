import { deployments, ethers, getNamedAccounts, network } from "hardhat";
import { getKeyHash, readAll, requestAll } from "./utils";
import hre from "hardhat";

async function main() {
  const networkName = hre.network.name;
  const ACC_ID =
    network.name == "baobab"
      ? (process.env.BAOBAB_ACC_ID as string)
      : (process.env.CYPRESS_ACC_ID as string);
  const explorerBaseUrl =
    network.name == "baobab"
      ? "https://baobab.klaytnfinder.io/tx"
      : "https://klaytnfinder.io/tx";

  const _inspectorConsumer = await deployments.get("InspectorConsumer");
  const inspectorConsumer = await ethers.getContractAt(
    _inspectorConsumer.abi,
    _inspectorConsumer.address
  );

  const {
    sResponse: rrResponseBefore,
    sRandomWord: vrfResponseBefore,
    rrRequestId: rrRequestIdBefore,
    vrfRequestId: vrfRequestIdBefore,
  } = await readAll(inspectorConsumer);
  await requestAll(ACC_ID, explorerBaseUrl, inspectorConsumer, networkName);
  await new Promise((resolve) => setTimeout(resolve, 5000));
  const {
    sResponse: rrResponseAfter,
    sRandomWord: vrfResponseAfter,
    rrRequestId: rrRequestIdAfter,
    vrfRequestId: vrfRequestIdAfter,
  } = await readAll(inspectorConsumer);

  if (
    rrResponseBefore == rrResponseAfter ||
    rrRequestIdBefore == rrRequestIdAfter
  ) {
    console.error("RR fulfillment: FAILURE");
  } else {
    console.log("RR fulfillment: SUCCESS");
  }

  if (
    vrfResponseBefore == vrfResponseAfter ||
    vrfRequestIdBefore == vrfRequestIdAfter
  ) {
    console.error("VRF fulfillment: FAILURE");
  } else {
    console.log("VRF fulfillment: SUCCESS");
  }
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
