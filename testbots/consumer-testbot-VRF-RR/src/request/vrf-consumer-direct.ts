import * as dotenv from "dotenv";
dotenv.config();
import { ethers } from "ethers";
import { existsSync } from "fs";
import { ILogData } from "../types";
import {
  getTimestampByBlock,
  readTextFile,
  writeTextAppend,
  writeTextFile,
} from "../utils";
import { buildWallet, sendTransaction } from "./utils";

const abis = await readTextFile("./src/abis/consumer.json");
const vrfAbis = await readTextFile("./src/abis/vrf-coordinator.json");

const VRF_CONSUMER = process.env.VRF_CONSUMER;
const KEY_HASH = process.env.KEY_HASH;

let jsonResult: ILogData[] = [];

export async function sendRequestRandomWordsDirect(date: string) {
  const iface = new ethers.utils.Interface(abis);
  const gasLimit = 3_000_000; // FIXME

  const callbackGasLimit = 500_000;
  const numWords = 2;
  const wallet = buildWallet();

  const jsonPath = `./tmp/request/requestRandomwords-${date}.json`;
  const errorPath = `./tmp/request/requestRandomwords-error-${date}.txt`;
  let fileData = "";
  if (existsSync(jsonPath)) fileData = await readTextFile(jsonPath);
  else jsonResult = [];

  await writeTextFile(jsonPath, JSON.stringify(jsonResult));
  if (fileData) jsonResult = <ILogData[]>JSON.parse(fileData);
  try {
    const payload = iface.encodeFunctionData("requestRandomWordsDirect", [
      KEY_HASH,
      callbackGasLimit,
      numWords,
    ]);
    const provider = new ethers.providers.JsonRpcProvider(
      process.env.PROVIDER_URL
    );
    const vrfCoordinator = new ethers.Contract(
      "0xfa605ca6dc9414e0f7fa322d3fd76535b33f7a4f",
      vrfAbis,
      provider
    );

    const value = await vrfCoordinator.estimateDirectPaymentFee();

    const tx = await sendTransaction(
      wallet,
      VRF_CONSUMER,
      payload,
      gasLimit,
      value
    );
    const txReceipt = await tx.wait();
    const requestObject = iface.parseLog(txReceipt.logs[4]).args;
    const requestedTime = await getTimestampByBlock(txReceipt.blockNumber);

    const result: ILogData = {
      block: txReceipt.blockNumber,
      txHash: txReceipt.transactionHash,
      requestId: requestObject.requestId.toString(),
      accId: requestObject.accId.toString(),
      isDirectPayment: requestObject.isDirectPayment,
      requestedTime,
    };
    jsonResult.push(result);
    console.log("VRF-Direct: Requested ", txReceipt.blockNumber);

    await writeTextFile(jsonPath, JSON.stringify(jsonResult));
  } catch (error) {
    console.log("VRF-Direct:", error);
    const d = new Date();
    await writeTextAppend(errorPath, `${d.toISOString()}:${error}\n`);
  }
}
