export class IcnError extends Error {
  constructor(public readonly code: IcnErrorCode, message?: string, public readonly value?: any) {
    super(message)
    this.name = IcnErrorCode[code]
    Object.setPrototypeOf(this, new.target.prototype)
  }
}

export enum IcnErrorCode {
  NonExistantEventError = 10000,
  InvalidOperator,
  MissingMnemonic,
  MissingJsonRpcProvider,
  MissingKeyInJson
}
