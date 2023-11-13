export class PorError extends Error {
  constructor(public readonly code: PorErrorCode, message?: string, public readonly value?) {
    super(message)
    this.name = PorErrorCode[code]
    this.value = value
    Object.setPrototypeOf(this, new.target.prototype)
  }
}

export enum PorErrorCode {
  IncompleteDataFeed,
  IndexOutOfBoundaries,
  InvalidAdapter,
  InvalidDataFeed,
  InvalidDataFeedFormat,
  InvalidReducer,
  MissingKeyInJson,
  UnexpectedNumberOfJobs
}
