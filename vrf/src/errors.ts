class VrfError extends Error {
  constructor(public readonly code: VrfErrorCode, message?: string, public readonly value?) {
    super(message)
    this.name = VrfErrorCode[code]
    this.value = value
    Object.setPrototypeOf(this, new.target.prototype)
  }
}

enum VrfErrorCode {
  InvalidProofError = 10000,
}

export { VrfError, VrfErrorCode }
