export enum SERVICE {
  VRF = "vrf",
  REQUEST_RESPONSE = "request-response",
  AGGREGATOR = "aggregator",
  FETCHER = "fetcher",
}

export enum QUEUE_STATUS {
  WAITING = "waiting",
  ACTIVE = "active",
  COMPLETED = "completed",
  FAILED = "failed",
  DELAYED = "delayed",
}

export enum QUEUE_ACTIVE_STATUS {
  START = "start",
  STOP = "stop",
}

export enum MONITOR_CONFIG {
  SLACK_URL = "slack_url",
  BALANCE_ALARM_AMOUNT = "balance_alarm_amount"
}