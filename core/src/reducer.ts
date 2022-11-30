export const reducerMapping = {
  PARSE: parseReducer,
  MUL: mulReducer
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

export function mulReducer(args: Number[]) {
  function wrapper(value: Number) {
    return args.reduce((acc, v) => Number(acc) * Number(v), value)
  }
  return wrapper
}
