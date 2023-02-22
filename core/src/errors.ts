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
  InvalidOperator,
  InvalidAdapter,
  InvalidAggregator,
  InvalidReducer,
  MissingMnemonic,
  MissingJsonRpcProvider,
  MissingKeyInJson,
  MissingAdapter,
  MissingAggregator,
  UniformWrongParams,
  InvalidListenerConfig,
  UndefinedListenerRequested,
  InvalidPriceFeed,
  InvalidPriceFeedFormat,
  MissingKeyValuePair,
  UnexpectedQueryOutput,
  TxInvalidAddress,
  TxProcessingResponseError,
  TxCannotEstimateGasError,
  ProviderNetworkError,
  InvalidDecodedMesssageLength,
  IndexOutOfBoundaries,
  AggregatorNotFound,
  AggregatorJobCanTakeMoreBreak,
  UndefinedAggregator,
  UnexpectedNumberOfJobsInQueue,
  TxNotMined
}
