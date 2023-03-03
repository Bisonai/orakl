import { ethers } from "ethers";
import { ILogData, IRRLogData, IRRReporterData } from "../types";
import { existsSync } from "fs";
import { readTextFile, writeTextFile } from "../utils";
let requestedNumber = 0;
let totalResponse = 0;
let jsonResult: IRRReporterData[] = [];
let minResponseTime = 0;
let maxResponseTime = 0;
export async function reportRR() {
  const d = new Date();
  d.setDate(d.getDate() - 1);
  const m = d.toISOString().split("T")[0];
  const jsonPath = `./tmp/reporter/rr-${m}.json`;
  const jsonSummaryPath = `./tmp/reporter/rr-summary-${m}.json`;
  console.log("time", m);
  const jsonRequestPath = `./tmp/request/request-response-${m}.json`;
  const jsonRequestDirectPath = `./tmp/request/request-response-direct-${m}.json`;

  const jsonResponsePath = `./tmp/listener/request-respone-fulfill-log-${m}.json`;
  if (!existsSync(jsonRequestPath) || !existsSync(jsonResponsePath)) return;
  const jsonRequest = await readTextFile(jsonRequestPath);
  const jsonResponse = await readTextFile(jsonResponsePath);
  const dataRequesteds = <ILogData[]>JSON.parse(jsonRequest);
  const dataResponseds = <IRRLogData[]>JSON.parse(jsonResponse);

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
      let response: string = "";
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
        response = responseInfor.response;
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
      const result: IRRReporterData = {
        requestBlock: rq?.block,
        requestId: rq?.requestId,
        requestTxHash: rq?.txHash,
        requestedTime: rq?.requestedTime,
        responseBlock,
        address,
        responseTxHash,
        response,
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
  console.log("rr: finish");
}
