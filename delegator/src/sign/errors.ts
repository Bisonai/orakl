export class DelegatorError extends Error {
  constructor(public readonly code: DelegatorErrorCode, message?: string, public readonly value?) {
    super(message)
    this.name = DelegatorErrorCode[code]
    this.value = value
    Object.setPrototypeOf(this, new.target.prototype)
  }
}

export enum DelegatorErrorCode {
  NotApprovedTransaction
}
