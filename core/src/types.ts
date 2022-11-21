export interface RequestEventData {
  specId: string
  requester: string
  payment: BigNumber
}

export interface DataFeedRequest {
  from: string
  specId: string
}
