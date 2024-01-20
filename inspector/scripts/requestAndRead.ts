import { deployments, ethers, getNamedAccounts, network } from "hardhat";
import { getKeyHash } from "./utils";

async function main() {
  const {
    sResponse: rrResponseBefore,
    sRandomWord: vrfResponseBefore,
    rrRequestId: rrRequestIdBefore,
    vrfRequestId: vrfRequestIdBefore,
  } = await read();

  await request();

  // wait 5 seconds for fulfillment and submission
  await new Promise((resolve) => setTimeout(resolve, 5000));
  const {
    sResponse: rrResponseAfter,
    sRandomWord: vrfResponseAfter,
    rrRequestId: rrRequestIdAfter,
    vrfRequestId: vrfRequestIdAfter,
  } = await read();

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

async function read() {
  const _inspectorConsumer = await deployments.get("InspectorConsumer");
  const inspectorConsumer = await ethers.getContractAt(
    _inspectorConsumer.abi,
    _inspectorConsumer.address
  );

  const sResponse = await inspectorConsumer.sResponse();
  const sRandomWord = await inspectorConsumer.sRandomWord();

  const rrRequestId = await inspectorConsumer.rrRequestId();
  const vrfRequestId = await inspectorConsumer.vrfRequestId();

  return { sResponse, sRandomWord, rrRequestId, vrfRequestId };
}

async function request() {
  let ACC_ID;
  let explorerBaseUrl;
  if (network.name == "baobab") {
    ACC_ID = process.env.BAOBAB_ACC_ID;
    explorerBaseUrl = "https://baobab.klaytnfinder.io/tx";
  } else {
    ACC_ID = process.env.CYPRESS_ACC_ID;
    explorerBaseUrl = "https://klaytnfinder.io/tx";
  }

  const _inspectorConsumer = await deployments.get("InspectorConsumer");
  const inspectorConsumer = await ethers.getContractAt(
    _inspectorConsumer.abi,
    _inspectorConsumer.address
  );

  const keyHash = getKeyHash();
  const callbackGasLimit = 500_000;
  const numWords = 1;

  const vrfTx = await (
    await inspectorConsumer.requestVRF(
      keyHash,
      ACC_ID,
      callbackGasLimit,
      numWords
    )
  ).wait();

  if (vrfTx.status == 1) {
    console.log("VRF request: SUCCESS");
    console.log(`${explorerBaseUrl}/${vrfTx.hash}`);
  } else {
    console.error("VRF request: FAILURE");
  }

  const rrTx = await (
    await inspectorConsumer.requestRR(ACC_ID, callbackGasLimit)
  ).wait();
  if (vrfTx.status == 1) {
    console.log("RR request: SUCCESS");
    console.log(`${explorerBaseUrl}/${rrTx.hash}`);
  } else {
    console.error("RR request: FAILURE");
  }
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
