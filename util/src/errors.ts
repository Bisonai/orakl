export class ReducerError extends Error {
  constructor(public readonly code: ReducerErrorCode, message?: string, public readonly value?) {
    super(message)
    this.name = ReducerErrorCode[code]
    this.value = value
    Object.setPrototypeOf(this, new.target.prototype)
  }
}

export enum ReducerErrorCode {
  InvalidReducer,
  IndexOutOfBoundaries,
  MissingKeyInJson,
  DivisionByZero,
  InvalidData,
  InvalidDataFormat,
}
