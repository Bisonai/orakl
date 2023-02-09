import axios from 'axios'
import { mean as Mean, median as Median, mode as Mode, sum as Sum } from 'mathjs'
import { IcnError, IcnErrorCode } from '../errors'

export async function get(input: string) {
  try {
    const out = (await axios.get(input)).data
    return out
  } catch (e) {}
}

export function path(json, path: string[]) {
  let v = json

  for (const p of path) {
    if (p in v) v = v[p]
    else throw new IcnError(IcnErrorCode.MissingKeyInJson)
  }

  return v
}

export function mul(input: number, arg: number) {
  return input * arg
}

export function div(input: number, arg: number) {
  return input / arg
}

export function index(input: number[], arg: number) {
  return input[arg]
}

export function mean(input: number[]) {
  return Mean(input)
}

export function median(input: number[]) {
  return Median(input)
}

export function mode(input: number[]) {
  return Mode(input)
}

export function sum(input: number[]) {
  return Sum(input)
}
