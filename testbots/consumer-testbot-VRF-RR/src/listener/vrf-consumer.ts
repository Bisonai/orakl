import { ethers } from "ethers";
import { Event } from "./event";
import { IListenerConfig, ILogData, IVRFLogData } from "../types";
import { existsSync } from "fs";
import { readTextFile, writeTextFile } from "../utils";

const abis = await readTextFile("./src/abis/consumer.json");
let eventCount = 0;
let jsonResult: IVRFLogData[] = [];
export function buildVrfListener(config: IListenerConfig) {
  new Event(processConsumerEvent, abis, config).listen();
}

function processConsumerEvent(iface: ethers.utils.Interface) {
  let fileData = "";

  async function wrapper(log) {
    const eventData = iface.parseLog(log).args;
    const d = new Date();
    const m = d.toISOString().split("T")[0];
    const jsonPath = `./tmp/listener/consumer-fulfill-log-${m}.json`;
    if (existsSync(jsonPath)) {
      fileData = await readTextFile(jsonPath);
      if (fileData && jsonResult.length == 0)
        jsonResult = <IVRFLogData[]>JSON.parse(fileData);
    } else {
      jsonResult = [];
    }

    if (eventData) {
      let requestedTime: number = 0;
      let respondedTime: number = 0;
      let totalResponseTime: number = 0;

      try {
        const jsonRequestRandomwordsPath = `./tmp/request/requestRandomwords-${m}.json`;
        const jsonRequestRandomwordsPathDirect = `./tmp/request/requestRandomwords-Direct-${m}.json`;
        const blockInfor = await log.getBlock();
        respondedTime = blockInfor.timestamp;
        let contents = "";
        let dataRequestedRandomwords: ILogData[];
        let requestInfor: ILogData | undefined;
        if (existsSync(jsonRequestRandomwordsPath)) {
          contents = await readTextFile(jsonRequestRandomwordsPath);
          dataRequestedRandomwords = <ILogData[]>JSON.parse(contents);
          requestInfor = dataRequestedRandomwords.find(
            (obj) => obj.requestId === eventData.requestId.toString()
          );
          if (!requestInfor && existsSync(jsonRequestRandomwordsPathDirect)) {
            console.log("direct random", requestInfor);
            contents = await readTextFile(jsonRequestRandomwordsPathDirect);
            dataRequestedRandomwords = <ILogData[]>JSON.parse(contents);
            requestInfor = dataRequestedRandomwords.find(
              (obj) => obj.requestId === eventData.requestId.toString()
            );
          }
          console.log(requestInfor);
          if (requestInfor && requestInfor.requestedTime > 0) {
            requestedTime = requestInfor.requestedTime;
            totalResponseTime = blockInfor.timestamp - requestedTime;
          }
        }
      } catch (error) {
        console.error(error);
      }
      const result: IVRFLogData = {
        block: log.blockNumber,
        address: log.address,
        txHash: log.transactionHash,
        requestId: eventData.requestId.toString(),
        randomWords: eventData.randomWords.map((r) => {
          return r.toString();
        }),
        requestedTime,
        respondedTime,
        totalRequestTime: totalResponseTime,
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
