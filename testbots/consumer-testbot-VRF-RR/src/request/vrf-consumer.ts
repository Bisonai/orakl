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
const VRF_CONSUMER = process.env.VRF_CONSUMER;
const ACC_ID = process.env.ACC_ID;
const KEY_HASH = process.env.KEY_HASH;
let jsonResult: ILogData[] = [];

export async function sendRequestRandomWords(date: string) {
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

  if (fileData) jsonResult = <ILogData[]>JSON.parse(fileData);
  await writeTextFile(jsonPath, JSON.stringify(jsonResult));

  try {
    const payload = iface.encodeFunctionData("requestRandomWords", [
      KEY_HASH,
      ACC_ID,
      callbackGasLimit,
      numWords,
    ]);
    const tx = await sendTransaction(wallet, VRF_CONSUMER, payload, gasLimit);
    const txReceipt = await tx.wait();
    const requestObject = iface.parseLog(txReceipt.logs[1]).args;

    const requestedTime = await getTimestampByBlock(txReceipt.blockNumber);
    if (txReceipt.status == true) {
      const result: ILogData = {
        block: txReceipt.blockNumber,
        txHash: txReceipt.transactionHash,
        requestId: requestObject.requestId.toString(),
        accId: requestObject.accId.toString(),
        isDirectPayment: requestObject.isDirectPayment,
        requestedTime,
      };
      jsonResult.push(result);
      console.log("VRF-Prepayment: requested ", txReceipt.blockNumber);
    }

    await writeTextFile(jsonPath, JSON.stringify(jsonResult));
  } catch (error) {
    console.log("VRF-Prepayment", error);
    const d = new Date();
    await writeTextAppend(errorPath, `${d.toISOString()}:${error}\n`);
  }
}
