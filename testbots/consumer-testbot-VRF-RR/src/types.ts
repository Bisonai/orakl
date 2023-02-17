export interface IListenerBlock {
  startBlock: number;
  filePath: string;
}

export interface IListenerConfig {
  address: string;
  eventName: string;
}

export interface ILogData {
  block: string;
  txHash: string;
  requestId: string;
  accId: number;
  isDirectPayment: boolean;
}
export interface IRRLogData {
  block: number;
  address: string;
  txHash: string;
  requestId: string;
  response: string;
}
export interface IVRFLogData {
  block: number;
  address: string;
  txHash: string;
  requestId: string;
  randomWords: string[];
}
