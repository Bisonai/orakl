export class ApiAdapterError extends Error {
  constructor(public readonly code: ApiAdapterErrorCode, message?: string, public readonly value?) {
    super(message)
    this.name = ApiAdapterErrorCode[code]
    Object.setPrototypeOf(this, new.target.prototype)
  }
}

export enum ApiAdapterErrorCode {
  UnmatchingHash
}
