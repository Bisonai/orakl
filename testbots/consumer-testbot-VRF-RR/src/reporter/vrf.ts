import { ethers } from "ethers";
import {
  IListenerConfig,
  ILogData,
  IVRFLogData,
  IVRFReporterData,
} from "../types";
import { existsSync } from "fs";
import { getTimestampByBlock, readTextFile, writeTextFile } from "../utils";
let requestedNumber = 0;
let totalResponse = 0;
let minResponseTime = 0;
let maxResponseTime = 0;
export async function reportVRF() {
  let jsonResult: IVRFReporterData[] = [];
  const d = new Date();
  const td = d.toISOString().split("T")[0];
  d.setDate(d.getDate() - 1);
  const m = d.toISOString().split("T")[0];
  const jsonPath = `./tmp/reporter/vrf-${m}.json`;
  const jsonSummaryPath = `./tmp/reporter/vrf-summary-${m}.json`;
  console.log("time", m);
  const jsonRequestPath = `./tmp/request/requestRandomwords-${m}.json`;
  const jsonRequestDirectPath = `./tmp/request/requestRandomwords-Direct-${m}.json`;

  const jsonResponsePath = `./tmp/listener/consumer-fulfill-log-${m}.json`;
  const jsonResponsePathToday = `./tmp/listener/consumer-fulfill-log-${td}.json`;

  if (!existsSync(jsonRequestPath) || !existsSync(jsonResponsePath)) return;
  const jsonRequest = await readTextFile(jsonRequestPath);
  const jsonResponse = await readTextFile(jsonResponsePath);
  const dataRequesteds = <ILogData[]>JSON.parse(jsonRequest);
  const dataResponseds = <IVRFLogData[]>JSON.parse(jsonResponse);
  if (existsSync(jsonResponsePathToday)) {
    const jsonResponseToday = await readTextFile(jsonResponsePathToday);
    const dataResponsedsToday = <IVRFLogData[]>JSON.parse(jsonResponseToday);
    if (dataResponsedsToday.length > 0)
      dataResponseds.push(...dataResponsedsToday);
  }

  if (existsSync(jsonRequestDirectPath)) {
    const requestDirects = await readTextFile(jsonRequestDirectPath);
    const dataRequestDirect = <ILogData[]>JSON.parse(requestDirects);
    if (dataRequestDirect.length > 0) dataRequesteds.push(...dataRequestDirect);
  }

  await Promise.all(
    dataRequesteds.map(async (rq) => {
      let responseBlock: number = 0;
      let address: string = "";
      let responseTxHash: string = "";
      let randomWords: string[] = [];
      let respondedTime: number = 0;
      let totalResponseTime: number = 0;
      let hasResponse: boolean = false;
      const responseInfor = dataResponseds.find(
        (f) => f.requestId === rq.requestId
      );
      if (responseInfor) {
        responseBlock = responseInfor.block;
        respondedTime = responseInfor.respondedTime;
        totalResponseTime = responseInfor.totalRequestTime;
        randomWords = responseInfor.randomWords;
        hasResponse = true;
        address = responseInfor.address;
        responseTxHash = responseInfor.txHash;
        totalResponse += 1;
        if (totalResponseTime > 0) {
          if (totalResponseTime < minResponseTime || minResponseTime === 0)
            minResponseTime = totalResponseTime;
          if (totalResponseTime > maxResponseTime)
            maxResponseTime = totalResponseTime;
        }
      }
      const result: IVRFReporterData = {
        requestBlock: rq?.block,
        requestId: rq?.requestId,
        requestTxHash: rq?.txHash,
        requestedTime: rq?.requestedTime,
        responseBlock,
        address,
        responseTxHash,
        randomWords,
        respondedTime,
        totalResponseTime,
        hasResponse,
      };
      requestedNumber++;
      console.debug(
        "reporter:data",
        jsonResult.length,
        "event:",
        requestedNumber
      );

      jsonResult.push(result);
    })
  );
  const summaryInfor = {
    totalRequest: requestedNumber,
    totalResponse,
    totalNoResponse: requestedNumber - totalResponse,
    minResponseTime,
    maxResponseTime,
  };
  await writeTextFile(jsonSummaryPath, JSON.stringify(summaryInfor, null, 2));
  await writeTextFile(jsonPath, JSON.stringify(jsonResult, null, 2));
  console.log("vrf:finish");
}
