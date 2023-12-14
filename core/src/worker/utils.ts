import { OraklError, OraklErrorCode } from '../errors'

export function uniform(a: number, b: number): number {
  if (a > b) {
    throw new OraklError(OraklErrorCode.UniformWrongParams)
  }
  return a + Math.round(Math.random() * (b - a))
}
