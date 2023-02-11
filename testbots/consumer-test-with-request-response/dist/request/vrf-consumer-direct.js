import * as dotenv from "dotenv";
dotenv.config();
import { ethers } from "ethers";
import { existsSync } from "fs";
import { readTextFile, writeTextFile } from "../utils";
import { buildWallet, sendTransaction } from "./utils";
const abis = await readTextFile("./src/abis/consumer.json");
const VRF_CONSUMER = process.env.VRF_CONSUMER;
const ACC_ID = process.env.ACC_ID;
let jsonResult = [];
const jsonPath = "./tmp/request/requestRandomwordsDirect.json";
if (!existsSync(jsonPath))
    await writeTextFile(jsonPath, JSON.stringify(jsonResult));
const data = await readTextFile(jsonPath);
if (data)
    jsonResult = JSON.parse(data);
async function sendRequestRandomWords(numberOfRequest) {
    const iface = new ethers.utils.Interface(abis);
    const gasLimit = 3000000;
    const keyHash = "0x47ede773ef09e40658e643fe79f8d1a27c0aa6eb7251749b268f829ea49f2024";
    const callbackGasLimit = 500000;
    const numWords = 2;
    const wallet = buildWallet();
    try {
        const payload = iface.encodeFunctionData("requestRandomWordsDirect", [
            keyHash,
            callbackGasLimit,
            numWords,
        ]);
        let requested = 0;
        const value = ethers.utils.parseEther("0.1");
        const requests = Array.from(Array(numberOfRequest).keys());
        await Promise.all(requests.map(async (request) => {
            try {
                const txReceipt = await sendTransaction(wallet, VRF_CONSUMER, payload, gasLimit, value);
                console.log("request randomword: ", request);
                requested += 1;
                const tx = await txReceipt.wait();
                const requestObject = iface.parseLog(tx.logs[4]).args;
                console.log("tx", requestObject);
                const result = {
                    requestId: requestObject.requestId.toString(),
                    accId: requestObject.accId.toString(),
                    isDirectPayment: requestObject.isDirectPayment,
                };
                jsonResult.push(result);
            }
            catch (error) {
                console.log("send request error", error);
            }
        }));
        await writeTextFile(jsonPath, JSON.stringify(jsonResult));
    }
    catch (error) {
        console.log(error);
    }
}
sendRequestRandomWords(1);
//# sourceMappingURL=vrf-consumer-direct.js.map