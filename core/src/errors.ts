export class IcnError extends Error {
  constructor(public readonly code: IcnErrorCode, message?: string, public readonly value?) {
    super(message)
    this.name = IcnErrorCode[code]
    this.value = value
    Object.setPrototypeOf(this, new.target.prototype)
  }
}

export enum IcnErrorCode {
  NonExistentEventError = 10000,
  AggregatorJobCanTakeMoreBreak,
  AggregatorNotFound,
  FailedToGetAggregate,
  FailedToGetAggregator,
  IncompleteDataFeed,
  IndexOutOfBoundaries,
  InvalidAdapter,
  InvalidAggregator,
  InvalidDataFeed,
  InvalidDataFeedFormat,
  InvalidDecodedMesssageLength,
  InvalidListenerConfig,
  InvalidOperator,
  InvalidReducer,
  MissingAdapter,
  MissingAggregator,
  MissingJsonRpcProvider,
  MissingKeyInJson,
  MissingKeyValuePair,
  MissingMnemonic,
  ProviderNetworkError,
  TxCannotEstimateGasError,
  TxInvalidAddress,
  TxNotMined,
  TxProcessingResponseError,
  UndefinedAggregator,
  UndefinedListenerRequested,
  UnexpectedNumberOfJobsInQueue,
  UnexpectedQueryOutput,
  UniformWrongParams
}
