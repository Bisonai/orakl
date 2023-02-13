import { ethers } from "ethers";
import { Event } from "./event";
import { IListenerConfig } from "../types";
import { existsSync } from "fs";
import { readTextFile, writeTextFile } from "../utils";

const abis = await readTextFile("./src/abis/consumer.json");
let eventCount = 0;
let jsonResult: any = [];
let fileData = "";
export function buildVrfListener(config: IListenerConfig) {
  new Event(processConsumerEvent, abis, config).listen();
}

function processConsumerEvent(iface: ethers.utils.Interface) {
  async function wrapper(log) {
    const eventData = iface.parseLog(log).args;

    const d = new Date();
    const m = d.toISOString().split("T")[0];
    const jsonPath = `./tmp/listener/consumer-fulfill-log-${m}.json`;
    if (existsSync(jsonPath)) fileData = await readTextFile(jsonPath);

    if (fileData && jsonResult.length == 0) jsonResult = JSON.parse(fileData);

    if (eventData) {
      const result = {
        block: log.blockNumber,
        address: log.address,
        txHash: log.transactionHash,
        requestId: eventData.requestId.toString(),
        randomWords: eventData.randomWords.map((r) => {
          return r.toString();
        }),
      };
      jsonResult.push(result);
      await writeTextFile(jsonPath, JSON.stringify(jsonResult, null, 2));
      eventCount++;
      console.debug(
        "processVrfEvent:data",
        jsonResult.length,
        "event:",
        eventCount
      );
    }
  }

  return wrapper;
}

async function main() {
  const listenersConfig: IListenerConfig = {
    address: process.env.VRF_CONSUMER ?? "",
    eventName: "RandomWordsFulfilled",
  };

  console.log(listenersConfig);
  const config = listenersConfig;
  buildVrfListener(config);
}
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
