export class FetcherError extends Error {
  constructor(public readonly code: FetcherErrorCode, message?: string, public readonly value?) {
    super(message)
    this.name = FetcherErrorCode[code]
    this.value = value
    Object.setPrototypeOf(this, new.target.prototype)
  }
}

export enum FetcherErrorCode {
  IndexOutOfBoundaries,
  InvalidAdapter,
  InvalidDataFeed,
  InvalidDataFeedFormat,
  InvalidReducer,
  MissingKeyInJson,
  UnexpectedNumberOfJobs,
  DivisionByZero,
}
