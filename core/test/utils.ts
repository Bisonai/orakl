import { jest } from '@jest/globals'
import { MockQueue } from '../src/types'

export const QUEUE: MockQueue = {
  add: jest.fn(),
  process: jest.fn(),
  on: jest.fn(),
}
