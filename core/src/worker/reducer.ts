export const reducerMapping = {
  PARSE: parse,
  MUL: mul,
  POW10: pow10,
  ROUND: round
}

export function parse(args: string[]) {
  function wrapper(obj) {
    for (const a of args) {
      obj = obj[a]
    }
    return obj
  }
  return wrapper
}

export function mul(args: number[]) {
  function wrapper(value: number) {
    return args.reduce((acc, v) => Number(acc) * Number(v), value)
  }
  return wrapper
}

export function pow10(args: number) {
  function wrapper(value: number) {
    return Number(Math.pow(10, args)) * value
  }
  return wrapper
}
export function round() {
  function wrapper(value: number) {
    return Math.round(value)
  }
  return wrapper
}
