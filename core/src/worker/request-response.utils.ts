import { ethers } from 'ethers'
import { Logger } from 'pino'
import { OraklError, OraklErrorCode } from '../errors'
import {
  IRequestResponseTransactionParameters,
  ITransactionParameters,
  RequestCommitmentRequestResponse,
} from '../types'

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
  [JOB_ID_BYTES]: 'fulfillDataRequestBytes',
}

export function buildTransaction(
  payloadParameters: IRequestResponseTransactionParameters,
  to: string,
  gasMinimum: number,
  iface: ethers.utils.Interface,
  logger: Logger,
): ITransactionParameters {
  const gasLimit = payloadParameters.callbackGasLimit + gasMinimum

  const fulfillDataRequestFn = JOB_ID_MAPPING[payloadParameters.jobId]
  if (fulfillDataRequestFn == undefined) {
    const msg = `Unknown jobId ${payloadParameters.jobId}`
    logger.error(msg)
    throw new OraklError(OraklErrorCode.UnknownRequestResponseJob, msg)
  }

  let response
  switch (payloadParameters.jobId) {
    case JOB_ID_UINT128:
    case JOB_ID_INT256:
      response = Math.floor(payloadParameters.response)
      break
    case JOB_ID_BOOL:
      if (payloadParameters.response.toLowerCase() == 'false') {
        response = false
      } else {
        response = Boolean(payloadParameters.response)
      }
      break
    case JOB_ID_STRING:
      response = String(payloadParameters.response)
      break
    case JOB_ID_BYTES32:
    case JOB_ID_BYTES:
      response = payloadParameters.response
      break
  }

  const rc: RequestCommitmentRequestResponse = [
    payloadParameters.blockNum,
    payloadParameters.accId,
    payloadParameters.numSubmission,
    payloadParameters.callbackGasLimit,
    payloadParameters.sender,
    payloadParameters.isDirectPayment,
    payloadParameters.jobId,
  ]
  logger.debug(rc, 'rc')

  const payload = iface.encodeFunctionData(fulfillDataRequestFn, [
    payloadParameters.requestId,
    response,
    rc,
  ])

  const tx = {
    payload,
    gasLimit,
    to,
  }
  logger.debug(tx, 'tx')

  return tx
}
