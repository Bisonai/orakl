import axios from 'axios'
import { Logger } from 'pino'
import { ORAKL_NETWORK_API_URL } from '../settings'
import { buildUrl } from '../utils'
import { OraklError, OraklErrorCode } from '../errors'
import { ISubmissionData } from './types'

export async function storeSubmission({
  submissionData,
  logger
}: {
  submissionData: ISubmissionData
  logger?: Logger
}) {
  try {
    const endpoint = buildUrl(ORAKL_NETWORK_API_URL, `last-submission/upsert`)
    const response = await axios.post(endpoint, { ...submissionData })
    logger?.info('Reporter submission upserted', response.data)
    return response.data
  } catch (e) {
    logger?.error(e.msg)
    throw new OraklError(OraklErrorCode.FailedToUpsertSubmission)
  }
}

export async function loadAggregatorByAddress({
  address,
  logger
}: {
  address: string
  logger?: Logger
}) {
  try {
    const endpoint = buildUrl(ORAKL_NETWORK_API_URL, `aggregator/${address}`)
    const response = await axios.get(endpoint)
    return response.data
  } catch (e) {
    logger?.error(e.msg)
    throw new OraklError(OraklErrorCode.FailedToLoadAggregator)
  }
}
