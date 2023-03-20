import { ethers } from "ethers";
import { Event } from "./event";
import { IListenerConfig, ILogData, IRRLogData } from "../types";
import { existsSync } from "fs";
import { getTimestampByBlock, readTextFile, writeTextFile } from "../utils";

const abis = await readTextFile(`./src/abis/request-response.json`);
let jsonResult: IRRLogData[] = [];
export function buildListener(config: IListenerConfig) {
  
  new Event(processConsumerEvent, abis, config).listen();
}

function processConsumerEvent(iface: ethers.utils.Interface) {
  let fileData = "";
  async function wrapper(log) {
    const eventData = iface.parseLog(log).args;

    const d = new Date();
    const m = d.toISOString().split("T")[0];
    const jsonPath = `./tmp/listener/request-respone-fulfill-log-${m}.json`;
    if (existsSync(jsonPath)) {
      fileData = await readTextFile(jsonPath);
      if (fileData && jsonResult.length == 0)
        jsonResult = <IRRLogData[]>JSON.parse(fileData);
    } else {
      jsonResult = [];
    }

    if (eventData) {
      let requestedTime: number = 0;
      let respondedTime: number = 0;
      let totalResponseTime: number = 0;

      try {
        const jsonRequestPath = `./tmp/request/request-response-${m}.json`;
        const jsonRequestPathDirect = `./tmp/request/request-response-direct-${m}.json`;
        const timestamp = await getTimestampByBlock(log.blockNumber);
        respondedTime = timestamp;
        let contents = "";
        let dataRequestedRandomwords: ILogData[];
        let requestInfor: ILogData | undefined;
        if (existsSync(jsonRequestPath)) {
          contents = await readTextFile(jsonRequestPath);
          dataRequestedRandomwords = <ILogData[]>JSON.parse(contents);
          requestInfor = dataRequestedRandomwords.find(
            (obj) => obj.requestId === eventData.requestId.toString()
          );
        }
        if (!requestInfor && existsSync(jsonRequestPathDirect)) {
          contents = await readTextFile(jsonRequestPathDirect);
          dataRequestedRandomwords = <ILogData[]>JSON.parse(contents);
          requestInfor = dataRequestedRandomwords.find(
            (obj) => obj.requestId === eventData.requestId.toString()
          );
        }
        if (requestInfor) {
          requestedTime = requestInfor.requestedTime;
          totalResponseTime = timestamp - requestedTime;
        }
      } catch (error) {
        console.error(error);
      }
      const result: IRRLogData = {
        block: log.blockNumber,
        address: log.address,
        txHash: log.transactionHash,
        requestId: eventData.requestId.toString(),
        response: eventData.response.toString(),
        requestedTime,
        respondedTime,
        totalRequestTime: totalResponseTime,
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
    eventName: "DataFulfilled",
  };

  console.log(listenersConfig);
  const config = listenersConfig;
  buildListener(config);
}
main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
