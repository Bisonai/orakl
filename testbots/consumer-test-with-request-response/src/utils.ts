import * as Fs from 'node:fs/promises'
import * as fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'

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

export function mkTmpFile({ fileName }: { fileName: string }): string {
  const appPrefix = 'orakl'
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), appPrefix))
  const tmpFilePath = path.join(tmpDir, fileName)
  return tmpFilePath
}

