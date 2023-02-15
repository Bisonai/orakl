import { ethers } from "ethers";
import { Event } from "./event";
import { IListenerConfig } from "../types";
import { existsSync } from "fs";
import { readTextFile, writeTextFile } from "../utils";

const abis = await readTextFile(`./src/abis/request-response.json`);
let jsonResult: any = [];
let fileData = "";
export function buildListener(config: IListenerConfig) {
  new Event(processConsumerEvent, abis, config).listen();
}

function processConsumerEvent(iface: ethers.utils.Interface) {
  async function wrapper(log) {
    const eventData = iface.parseLog(log).args;

    const d = new Date();
    const m = d.toISOString().split("T")[0];
    const jsonPath = `./tmp/listener/request-respone-fulfill-log-${m}.json`;
    if (existsSync(jsonPath)) fileData = await readTextFile(jsonPath);

    if (fileData && jsonResult.length == 0) jsonResult = JSON.parse(fileData);
    if (eventData) {
      const result = {
        block: log.blockNumber,
        address: log.address,
        txHash: log.transactionHash,
        requestId: eventData.requestId.toString(),
        response: eventData.response.toString(),
      };
      jsonResult.push(result);
      await writeTextFile(jsonPath, JSON.stringify(jsonResult));
      console.debug("processEvent:data", jsonResult.length);
    }
  }

  return wrapper;
}

async function main() {
  const listenersConfig: IListenerConfig = {
    address: process.env.RR_CONSUMER ?? "",
    eventName: "DataFulfilled"
  };

  console.log(listenersConfig);
  const config = listenersConfig;
  buildListener(config);
}
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
