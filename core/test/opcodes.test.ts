import { describe, test, expect } from '@jest/globals'
import { mul, div } from '../src/worker/opcodes'

describe('Opcodes', function () {
  test('Mul', function () {
    const input = 2
    const arg = 3
    expect(mul(input, arg)).toBe(6)
  })

  test('Div', function () {
    const input = 6
    const arg = 3
    expect(div(input, arg)).toBe(2)
  })
})
