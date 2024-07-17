import { ReducerError, ReducerErrorCode } from './errors'

export const REDUCER_MAPPING = {
  PATH: parseFn,
  PARSE: parseFn,
  MUL: mulFn,
  POW10: pow10Fn,
  ROUND: roundFn,
  INDEX: indexFn,
  DIV: divFn,
  DIVFROM: divFromFn,
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
      else throw new ReducerError(ReducerErrorCode.MissingKeyInJson)
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

export function divFromFn(args: number) {
  function wrapper(value: number) {
    if (value == 0) {
      throw new ReducerError(ReducerErrorCode.DivisionByZero)
    }
    return args / value
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
    throw new ReducerError(ReducerErrorCode.IndexOutOfBoundaries)
  }

  function wrapper(obj) {
    if (args >= obj.length) {
      throw new ReducerError(ReducerErrorCode.IndexOutOfBoundaries)
    } else {
      return obj[args]
    }
  }
  return wrapper
}

export function buildReducer(reducerMapping, reducers) {
  return reducers.map((r) => {
    const reducer = reducerMapping[r.function.toUpperCase()]
    if (!reducer) {
      throw new ReducerError(ReducerErrorCode.InvalidReducer)
    }
    return reducer(r?.args)
  })
}

// https://medium.com/javascript-scene/reduce-composing-software-fe22f0c39a1d
export const pipe =
  (...fns) =>
  (x) =>
    fns.reduce((v, f) => f(v), x)

export function checkDataFormat(data) {
  if (!data) {
    throw new ReducerError(ReducerErrorCode.InvalidData)
  } else if (!Number.isInteger(data)) {
    throw new ReducerError(ReducerErrorCode.InvalidDataFormat)
  }
}
