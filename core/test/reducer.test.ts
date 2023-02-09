import { describe, test, expect } from '@jest/globals'
import { parseFn, mulFn, pow10Fn, roundFn } from '../src/worker/reducer'

describe('Reducers', function () {
  test('parseFn with array input', function () {
    const obj = {
      RAW: { ETH: { USD: { PRICE: 123 } } },
      DISPLAY: { ETH: { USD: [Object] } }
    }
    const fn = parseFn(['RAW', 'ETH', 'USD', 'PRICE'])
    fn(obj)
    expect(fn(obj)).toBe(123)
  })

  test('parseFn with string input', function () {
    const obj = {
      RAW: { ETH: { USD: { PRICE: 123 } } },
      DISPLAY: { ETH: { USD: [Object] } }
    }
    const fn = parseFn('RAW,ETH,USD,PRICE')
    fn(obj)
    expect(fn(obj)).toBe(123)
  })

  test('Mul', function () {
    expect(mulFn(2)(3)).toBe(6)
  })

  test('Pow10', function () {
    expect(pow10Fn(4)(1)).toBe(10_000)
    expect(pow10Fn(4)(2)).toBe(20_000)
  })

  test('Round', function () {
    expect(roundFn()(1.1)).toBe(1)
    expect(roundFn()(1.5)).toBe(2)
    expect(roundFn()(1.9)).toBe(2)
  })
})
