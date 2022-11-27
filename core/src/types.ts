import { BigNumber } from 'ethers'

export interface RequestEventData {
  specId: string
  requester: string
  payment: BigNumber
}

export interface DataFeedRequest {
  from: string
  specId: string
}

export interface IListeners {
  VRF: string[]
  AGGREGATORS: string[]
}

export interface ILog {
  address: string
  blockHash: string
  blockNumber: string
  data: string
  logIndex: string
  removed: boolean
  topics: string[]
  transactionHash: string
  transactionIndex: string
}

interface IHeader {
  'Content-Type': string
}

interface IReducer {
  function: string
  args: string[]
}

interface IFeed {
  url: string
  method: string
  headers?: IHeader[]
  reducers?: IReducer[]
}

export interface IAdapter {
  active: boolean
  name: string
  job_type: string
  adapter_id: string
  oracle: string
  feeds: IFeed[]
}
