import { ethers } from 'ethers'
import { IL2DataRequestFulfilled } from '../types'

const UINT128 = ethers.utils.id('uint128')
const INT256 = ethers.utils.id('int256')
const BOOL = ethers.utils.id('bool')
const STRING = ethers.utils.id('string')
const BYTES32 = ethers.utils.id('bytes32')
const BYTES = ethers.utils.id('bytes')

export const parseResponse = {
  [UINT128]: function (x: IL2DataRequestFulfilled) {
    return x.responseUint128.toString()
  },
  [INT256]: function (x: IL2DataRequestFulfilled) {
    return x.responseInt256.toString()
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
  },
} satisfies Record<string, (x: IL2DataRequestFulfilled) => number | string | boolean>
