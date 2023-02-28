import { type } from "os";

export interface IListenerBlock {
  startBlock: number;
  filePath: string;
}

export interface IListenerConfig {
  address: string;
  eventName: string;
}

export interface ILogData {
  block: number;
  txHash: string;
  requestId: string;
  accId: number;
  isDirectPayment: boolean;
  requestedTime: number;
}
export interface IRRLogData {
  block: number;
  address: string;
  txHash: string;
  requestId: string;
  response: string;
  respondedTime: number;
  requestedTime: number;
  totalRequestTime: number;
}

export interface IVRFLogData {
  block: number;
  address: string;
  txHash: string;
  requestId: string;
  randomWords: string[];
  respondedTime: number;
  requestedTime: number;
  totalRequestTime: number;
}

export interface IVRFReporterData {
  requestBlock: number | undefined;
  responseBlock: number | undefined;
  address: string | undefined;
  RequestTxHash: string | undefined;
  ResponseTxHash: string | undefined;
  requestId: string | undefined;
  randomWords: string[] | undefined;
  respondedTime: number | undefined;
  requestedTime: number | undefined;
  totalResponseTime: number | undefined;
  hasResponse: boolean | undefined;
}
