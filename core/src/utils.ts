import * as Fs from 'node:fs/promises'
import * as fs from 'node:fs'
import { IcnError, IcnErrorCode } from './errors'

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

export async function sendTransaction(wallet, to, payload, gasLimit?, value?) {
  const tx = {
    from: wallet.address,
    to: to,
    data: add0x(payload),
    gasLimit: gasLimit || '0x34710', // FIXME
    value: value || '0x00'
  }
  console.debug('sendTransaction:tx')
  console.debug(tx)

  const txReceipt = await wallet.sendTransaction(tx)
  console.debug('sendTransaction:txReceipt')
  console.debug(txReceipt)
}

export function mkdir(dir: string) {
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true })
  }
}

export async function readTextFile(filepath: string) {
  return await Fs.readFile(filepath, 'utf8')
}

export async function writeTextFile(filepath: string, content: string) {
  await Fs.writeFile(filepath, content)
}
