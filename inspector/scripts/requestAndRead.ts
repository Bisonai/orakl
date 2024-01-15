import { deployments, ethers, getNamedAccounts } from "hardhat";
import { getKeyHash } from "./utils";

const ACC_ID = process.env.ACC_ID;

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
    throw "check if request response is alive";
  }
  if (
    vrfResponseBefore == vrfResponseAfter ||
    vrfRequestIdBefore == vrfRequestIdAfter
  ) {
    throw "check if vrf is alive";
  }

  console.log("successful");
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
  const _inspectorConsumer = await deployments.get("InspectorConsumer");
  const inspectorConsumer = await ethers.getContractAt(
    _inspectorConsumer.abi,
    _inspectorConsumer.address
  );

  const keyHash = getKeyHash();
  const callbackGasLimit = 500_000;
  const numWords = 1;

  await (
    await inspectorConsumer.requestVRF(
      keyHash,
      ACC_ID,
      callbackGasLimit,
      numWords
    )
  ).wait();

  console.log("Requested random words");

  await (await inspectorConsumer.requestRR(ACC_ID, callbackGasLimit)).wait();

  console.log("Requested data");
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
