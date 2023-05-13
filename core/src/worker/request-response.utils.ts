import { ethers } from 'ethers'

export const JOB_ID_UINT128 = ethers.utils.id('uint128')
export const JOB_ID_INT256 = ethers.utils.id('int256')
export const JOB_ID_BOOL = ethers.utils.id('bool')
export const JOB_ID_STRING = ethers.utils.id('string')
export const JOB_ID_BYTES32 = ethers.utils.id('bytes32')
export const JOB_ID_BYTES = ethers.utils.id('bytes')

export const JOB_ID_MAPPING = {
  [JOB_ID_UINT128]: 'fulfillDataRequestUint128',
  [JOB_ID_INT256]: 'fulfillDataRequestInt256',
  [JOB_ID_BOOL]: 'fulfillDataRequestBool',
  [JOB_ID_STRING]: 'fulfillDataRequestString',
  [JOB_ID_BYTES32]: 'fulfillDataRequestBytes32',
  [JOB_ID_BYTES]: 'fulfillDataRequestBytes'
}
