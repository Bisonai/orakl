import { Logger } from 'pino'
import { OraklError, OraklErrorCode } from '../errors'

export function buildReducer(reducerMapping, reducers) {
  return reducers.map((r) => {
    const reducer = reducerMapping[r.function]
    if (!reducer) {
      throw new OraklError(OraklErrorCode.InvalidReducer)
    }
    return reducer(r?.args)
  })
}

export function uniform(a: number, b: number): number {
  if (a > b) {
    throw new OraklError(OraklErrorCode.UniformWrongParams)
  }
  return a + Math.round(Math.random() * (b - a))
}
