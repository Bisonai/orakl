import { readFile } from 'node:fs/promises'

export async function loadJson(filepath: string) {
  const json = await readFile(filepath, 'utf8')
  return JSON.parse(json)
}
