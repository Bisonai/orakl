import dotenv from "dotenv";
import { Contract } from "ethers";

dotenv.config();

export function getKeyHash(networkName: string) {
  if (networkName == "baobab") {
    return "0xd9af33106d664a53cb9946df5cd81a30695f5b72224ee64e798b278af812779c";
  } else if (networkName == "cypress") {
    return "0x6cff5233743b3c0321a19ae11ab38ae0ddc7ddfe1e91b162fa8bb657488fb157";
  } else {
    throw new Error(`Key Hash is not defined for network: ${networkName}`);
  }
}

export async function readAll(inspectorConsumer: Contract) {
  const { sRandomWord, vrfRequestId } = await readVrf(inspectorConsumer);
  const { sResponse, rrRequestId } = await readRr(inspectorConsumer);
  return { sResponse, sRandomWord, rrRequestId, vrfRequestId };
}

export async function readVrf(inspectorConsumer: Contract) {
  const sRandomWord = await inspectorConsumer.sRandomWord();
  const vrfRequestId = await inspectorConsumer.vrfRequestId();
  return { sRandomWord, vrfRequestId };
}

export async function readRr(inspectorConsumer: Contract) {
  const sResponse = await inspectorConsumer.sResponse();
  const rrRequestId = await inspectorConsumer.rrRequestId();
  return { sResponse, rrRequestId };
}

export async function requestAll(
  accId: string,
  explorerBaseUrl: string,
  inspectorConsumer: Contract,
  networkName: string
) {
  await requestVrf(accId, explorerBaseUrl, inspectorConsumer, networkName);
  await requestRr(accId, explorerBaseUrl, inspectorConsumer);
}

export async function requestVrf(
  accId: string,
  explorerBaseUrl: string,
  inspectorConsumer: Contract,
  networkName: string
) {
  const keyHash = getKeyHash(networkName);
  const callbackGasLimit = 500_000;
  const numWords = 1;

  const vrfTx = await (
    await inspectorConsumer.requestVRF(
      keyHash,
      accId,
      callbackGasLimit,
      numWords
    )
  ).wait();

  if (vrfTx.status == 1) {
    console.log("VRF request: SUCCESS");
  } else {
    console.error("VRF request: FAILURE");
  }
  console.log(`${explorerBaseUrl}/${vrfTx.hash}`);
}

export async function requestRr(
  accId: string,
  explorerBaseUrl: string,
  inspectorConsumer: Contract
) {
  const callbackGasLimit = 500_000;

  const rrTx = await (
    await inspectorConsumer.requestRR(accId, callbackGasLimit)
  ).wait();

  if (rrTx.status == 1) {
    console.log("RR request: SUCCESS");
    console.log(`${explorerBaseUrl}/${rrTx.hash}`);
  } else {
    console.error("RR request: FAILURE");
  }
}
