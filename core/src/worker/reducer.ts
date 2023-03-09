import { OraklError, OraklErrorCode } from '../errors'

export const dataFeedReducerMapping = {
  PARSE: parseFn,
  MUL: mulFn,
  POW10: pow10Fn,
  ROUND: roundFn,
  INDEX: indexFn
}

export const requestResponseReducerMapping = {
  path: parseFn,
  mul: mulFn,
  div: divFn,
  pow10: pow10Fn,
  round: roundFn,
  index: indexFn
}

/**
 * Access data in JSON based on given path.
 *
 * Example
 * let obj = {
 *     RAW: { ETH: { USD: { PRICE: 123 } } },
 *     DISPLAY: { ETH: { USD: [Object] } }
 * }
 * const fn = parseFn(['RAW', 'ETH', 'USD', 'PRICE'])
 * fn(obj) // return 123
 */
export function parseFn(args: string | string[]) {
  if (typeof args == 'string') {
    args = args.split(',')
  }

  function wrapper(obj) {
    for (const a of args) {
      if (a in obj) obj = obj[a]
      else throw new OraklError(OraklErrorCode.MissingKeyInJson)
    }
    return obj
  }
  return wrapper
}

export function mulFn(args: number) {
  function wrapper(value: number) {
    return value * args
  }
  return wrapper
}

export function divFn(args: number) {
  function wrapper(value: number) {
    return value / args
  }
  return wrapper
}

export function pow10Fn(args: number) {
  function wrapper(value: number) {
    return Number(Math.pow(10, args)) * value
  }
  return wrapper
}

export function roundFn() {
  function wrapper(value: number) {
    return Math.round(value)
  }
  return wrapper
}

export function indexFn(args: number) {
  if (args < 0) {
    throw new OraklError(OraklErrorCode.IndexOutOfBoundaries)
  }

  function wrapper(obj) {
    if (args >= obj.length) {
      throw new OraklError(OraklErrorCode.IndexOutOfBoundaries)
    } else {
      return obj[args]
    }
  }
  return wrapper
}
