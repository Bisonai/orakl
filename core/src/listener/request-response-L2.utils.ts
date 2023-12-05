import { ethers } from 'ethers'
import { IL2DataRequestFulfilled } from '../types'

export const UINT128 = ethers.utils.id('uint128')
export const INT256 = ethers.utils.id('uint128')
export const BOOL = ethers.utils.id('bool')
export const STRING = ethers.utils.id('string')
export const BYTES32 = ethers.utils.id('bytes32')
export const BYTES = ethers.utils.id('bytes')

export const parseResponse = {
  [UINT128]: function (x: IL2DataRequestFulfilled) {
    return Number(x.responseUint128)
  },
  [INT256]: function (x: IL2DataRequestFulfilled) {
    return Number(x.responseInt256)
  },
  [BOOL]: function (x: IL2DataRequestFulfilled) {
    return x.responseBool
  },
  [STRING]: function (x: IL2DataRequestFulfilled) {
    return x.responseString
  },
  [BYTES32]: function (x: IL2DataRequestFulfilled) {
    return x.responseBytes32
  },
  [BYTES]: function (x: IL2DataRequestFulfilled) {
    return x.responseBytes
  }
} satisfies Record<string, (x: IL2DataRequestFulfilled) => number | string | boolean>
