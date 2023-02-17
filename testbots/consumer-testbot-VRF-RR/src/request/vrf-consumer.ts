import * as dotenv from "dotenv";
dotenv.config();
import { ethers } from "ethers";
import { existsSync } from "fs";
import { readTextFile, writeTextAppend, writeTextFile } from "../utils";
import { buildWallet, sendTransaction } from "./utils";

const abis = await readTextFile("./src/abis/consumer.json");
const VRF_CONSUMER = process.env.VRF_CONSUMER;
const ACC_ID = process.env.ACC_ID;
let jsonResult: any = [];

export async function sendRequestRandomWords() {
  const iface = new ethers.utils.Interface(abis);
  const gasLimit = 3_000_000; // FIXME
  const keyHash =
    "0x47ede773ef09e40658e643fe79f8d1a27c0aa6eb7251749b268f829ea49f2024";
  const callbackGasLimit = 500_000;
  const numWords = 2;
  const wallet = buildWallet();
  const d = new Date();
  const m = d.toISOString().split("T")[0];
  const jsonPath = `./tmp/request/requestRandomwords-${m}.json`;
  const errorPath = `./tmp/request/requestRandomwords-error-${m}.json`;

  let fileData = "";
  if (existsSync(jsonPath)) fileData = await readTextFile(jsonPath);
  await writeTextFile(jsonPath, JSON.stringify(jsonResult));

  if (fileData) jsonResult = JSON.parse(fileData);
  try {
    const payload = iface.encodeFunctionData("requestRandomWords", [
      keyHash,
      ACC_ID,
      callbackGasLimit,
      numWords,
    ]);
    let requested = 0;
    const txReceipt = await sendTransaction(
      wallet,
      VRF_CONSUMER,
      payload,
      gasLimit
    );
    requested += 1;
    const tx = await txReceipt.wait();
    const requestObject = iface.parseLog(tx.logs[1]).args;

    if (tx.status == true) {
      const result = {
        block: tx.blockNumber,
        txHash: tx.transactionHash,
        requestId: requestObject.requestId.toString(),
        accId: requestObject.accId.toString(),
        isDirectPayment: requestObject.isDirectPayment,
      };
      jsonResult.push(result);
      console.log("Requested: ", tx.blockNumber);
    }

    await writeTextFile(jsonPath, JSON.stringify(jsonResult));
  } catch (error) {
    console.log(error);
    await writeTextAppend(errorPath, `${d.toISOString()}:${error}\n`);
  }
}
