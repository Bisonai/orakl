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
  UniformWrongParams,
  InvalidListenerConfig,
  UndefinedListenerRequested,
  InvalidPriceFeed,
  InvalidPriceFeedFormat,
  MissingKeyValuePair,
<<<<<<< HEAD
  UnexpectedQueryOutput,
  TxInvalidAddress,
  TxProcessingResponseError,
  TxCannotEstimateGasError,
  ProviderNetworkError
||||||| parent of 44b0629 (feat: add InvalidDecodedMessageLength eror)
  UnexpectedQueryOutput
=======
  UnexpectedQueryOutput,
  InvalidDecodedMesssageLength
>>>>>>> 44b0629 (feat: add InvalidDecodedMessageLength eror)
}
