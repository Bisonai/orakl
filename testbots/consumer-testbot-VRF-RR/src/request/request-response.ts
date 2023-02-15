import * as dotenv from "dotenv";
dotenv.config();
import { ethers } from "ethers";
import { existsSync } from "fs";
import { readTextFile, writeTextAppend, writeTextFile } from "../utils";
import { buildWallet, sendTransaction } from "./utils";

const abis = await readTextFile("./src/abis/request-response.json");
const RR_CONSUMER = process.env.RR_CONSUMER;
const ACC_ID = process.env.ACC_ID;
let jsonResult: any = [];
export async function sendRequestData() {
  const iface = new ethers.utils.Interface(abis);
  const gasLimit = 3_000_000; // FIXME
  const callbackGasLimit = 500_000;
  const wallet = buildWallet();
  const d = new Date();
  const m = d.toISOString().split("T")[0];
  const jsonPath = `./tmp/request/request-response-${m}.json`;
  const errorPath = `./tmp/request/request-response-error-${m}.json`;

  let fileData = "";
  if (existsSync(jsonPath)) fileData = await readTextFile(jsonPath);
  await writeTextFile(jsonPath, JSON.stringify(jsonResult));
  try {
    const payload = iface.encodeFunctionData("requestData", [
      ACC_ID,
      callbackGasLimit,
    ]);
    const txReceipt = await sendTransaction(
      wallet,
      RR_CONSUMER,
      payload,
      gasLimit
    );
    const tx = await txReceipt.wait();
    const requestObject = iface.parseLog(tx.logs[1]).args;
    console.log("tx", requestObject);

    const result = {
      block: tx.blockNumber,
      txHash: tx.transactionHash,
      requestId: requestObject.requestId.toString(),
      accId: requestObject.accId.toString(),
      isDirectPayment: requestObject.isDirectPayment,
    };
    jsonResult.push(result);
    console.log("Requested: ", tx.blockNumber);

    await writeTextFile(jsonPath, JSON.stringify(jsonResult));
  } catch (error) {
    console.log(error);
    await writeTextAppend(errorPath, `${d.toISOString()}:${error}\n`);
  }
}
