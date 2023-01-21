export class CliError extends Error {
  constructor(public readonly code: CliErrorCode, message?: string, public readonly value?) {
    super(message)
    this.name = CliErrorCode[code]
    Object.setPrototypeOf(this, new.target.prototype)
  }
}

export enum CliErrorCode {
  NonExistantChain = 10000
}
