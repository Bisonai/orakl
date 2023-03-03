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

const abis = await readTextFile("./src/abis/request-response.json");
const rrAbis = await readTextFile("./src/abis/rr-coordinator.json");

const RR_CONSUMER = process.env.RR_CONSUMER;
let jsonResult: ILogData[] = [];

export async function sendRequestDataDirect() {
  const iface = new ethers.utils.Interface(abis);
  const gasLimit = 3_000_000; // FIXME
  const callbackGasLimit = 500_000;
  const wallet = buildWallet();
  const d = new Date();
  const m = d.toISOString().split("T")[0];
  const jsonPath = `./tmp/request/request-response-${m}.json`;
  const errorPath = `./tmp/request/request-response-error-${m}.txt`;

  let fileData = "";
  if (existsSync(jsonPath)) fileData = await readTextFile(jsonPath);
  if (fileData) jsonResult = <ILogData[]>JSON.parse(fileData);

  try {
    const payload = iface.encodeFunctionData("requestDataDirectPayment", [
      callbackGasLimit,
    ]);
    const provider = new ethers.providers.JsonRpcProvider(
      process.env.PROVIDER_URL
    );
    const rrCoordinator = new ethers.Contract(
      "0x402ab86A36686980F47C7097483d3ff1EAd5efE9",
      rrAbis,
      provider
    );

    const value = await rrCoordinator.estimateDirectPaymentFee();
    const tx = await sendTransaction(
      wallet,
      RR_CONSUMER,
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
    console.log("RR-Direct: Requested ", txReceipt.blockNumber);

    await writeTextFile(jsonPath, JSON.stringify(jsonResult));
  } catch (error) {
    console.log("RR-Direct", error);
    await writeTextAppend(errorPath, `${d.toISOString()}:${error}\n`);
  }
}
