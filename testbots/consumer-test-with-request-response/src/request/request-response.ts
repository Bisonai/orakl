import * as dotenv from "dotenv";
dotenv.config();
import { ethers } from "ethers";
import { existsSync } from "fs";
import { readTextFile, writeTextFile } from "../utils";
import { buildWallet, sendTransaction } from "./utils";

const abis = await readTextFile("./src/abis/request-response.json");
const RR_CONSUMER = process.env.RR_CONSUMER;
const ACC_ID = process.env.ACC_ID;
let jsonResult: any = [];
const jsonPath = "./tmp/request/request-response.json";
if (!existsSync(jsonPath))
  await writeTextFile(jsonPath, JSON.stringify(jsonResult));
const data = await readTextFile(jsonPath);
if (data) jsonResult = JSON.parse(data);

async function sendRequestRandomWords(numberOfRequest: number) {
  const iface = new ethers.utils.Interface(abis);
  const gasLimit = 3_000_000; // FIXME
  const callbackGasLimit = 500_000;
  const wallet = buildWallet();
  try {
    const payload = iface.encodeFunctionData("requestData", [
      ACC_ID,
      callbackGasLimit,
    ]);
    let requested = 0;
    const requests = Array.from(Array(numberOfRequest).keys());
    await Promise.all(
      requests.map(async (request) => {
        try {
          const txReceipt = await sendTransaction(
            wallet,
            RR_CONSUMER,
            payload,
            gasLimit
          );
          console.log("request data: ", request);
          requested += 1;
          const tx = await txReceipt.wait();
          const requestObject = iface.parseLog(tx.logs[1]).args;
          console.log("tx", requestObject);

          const result = {
            requestId: requestObject.requestId.toString(),
            accId: requestObject.accId.toString(),
            isDirectPayment: requestObject.isDirectPayment,
          };
          jsonResult.push(result);
        } catch (error) {
          console.log("send request error", error);
        }
      })
    );

    await writeTextFile(jsonPath, JSON.stringify(jsonResult));
  } catch (error) {
    console.log(error);
  }
}

sendRequestRandomWords(1);
