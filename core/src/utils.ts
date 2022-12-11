import * as Fs from 'node:fs/promises'
import { IcnError, IcnErrorCode } from './errors'
import { REDIS_HOST, REDIS_PORT } from './load-parameters'

export function buildBullMqConnection() {
  // FIXME Move to separate factory file?
  return {
    connection: {
      host: REDIS_HOST,
      port: REDIS_PORT
    }
  }
}

// FIXME create a settings file instead
// TODO move adapter out of src directory
export function buildAdapterRootDir() {
  return './src/adapter/'
}

export async function loadJson(filepath) {
  const json = await Fs.readFile(filepath, 'utf8')
  return JSON.parse(json)
}

// https://medium.com/javascript-scene/reduce-composing-software-fe22f0c39a1d
export const pipe =
  (...fns) =>
  (x) =>
    fns.reduce((v, f) => f(v), x)

/**
 * Access data in JSON based on given path.
 *
 * Example
 * let json = {
 *     RAW: { ETH: { USD: { PRICE: 123 } } },
 *     DISPLAY: { ETH: { USD: [Object] } }
 * }
 * readFromJson(json, ['RAW', 'ETH', 'USD', 'PRICE']) // return 123
 */
export function readFromJson(json, path: string[]) {
  let v = json

  for (const p of path) {
    if (p in v) v = v[p]
    else throw new IcnError(IcnErrorCode.MissingKeyInJson)
  }

  return v
}

export function remove0x(s) {
  if (s.substring(0, 2) == '0x') {
    return s.substring(2)
  }
}

export function add0x(s) {
  if (s.substring(0, 2) == '0x') {
    return s
  } else {
    return '0x' + s
  }
}

export function pad32Bytes(data) {
  data = remove0x(data)
  let s = String(data)
  while (s.length < (64 || 2)) {
    s = '0' + s
  }
  return s
}
