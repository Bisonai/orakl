export const reducerMapping = {
  PARSE: parseReducer,
  MUL: mulReducer,
  POW10: pow10Reducer,
  ROUND: roundReducer
}

export function parseReducer(args: string[]) {
  function wrapper(obj) {
    for (const a of args) {
      obj = obj[a]
    }

    return obj
  }

  return wrapper
}

export function mulReducer(args: number[]) {
  function wrapper(value: number) {
    return args.reduce((acc, v) => Number(acc) * Number(v), value)
  }
  return wrapper
}

export function pow10Reducer(args: number) {
  function wrapper(value: number) {
    return Number(Math.pow(10, args)) * value
  }
  return wrapper
}
export function roundReducer() {
  function wrapper(value: number) {
    return Math.round(value)
  }
  return wrapper
}
