import { IData } from '@bisonai/orakl-util'
import axios from 'axios'
import { Logger } from 'pino/pino'
import { OraklError, OraklErrorCode } from '../errors'
import { CHAIN, ORAKL_NETWORK_API_URL } from '../settings'
import { IAggregator } from '../types'
import { buildUrl } from '../utils'

export async function loadAggregator({
  aggregatorHash,
  logger
}: {
  aggregatorHash: string
  logger: Logger
}) {
  try {
    const url = buildUrl(ORAKL_NETWORK_API_URL, `aggregator/${aggregatorHash}/${CHAIN}`)
    const aggregator: IAggregator = (await axios.get(url))?.data
    return aggregator
  } catch (e) {
    logger.error(`Failed to load aggregator with :${aggregatorHash}`)
    throw new OraklError(OraklErrorCode.FailedToGetAggregator)
  }
}

export async function insertData({
  aggregatorId,
  feedId,
  value,
  logger
}: {
  aggregatorId: bigint
  feedId: bigint
  value: number
  logger: Logger
}) {
  const timestamp = new Date(Date.now()).toString()
  const data: IData[] = [
    {
      aggregatorId: aggregatorId.toString(),
      feedId,
      timestamp,
      value
    }
  ]

  try {
    const url = buildUrl(ORAKL_NETWORK_API_URL, 'data')
    const response = await axios.post(url, { data })

    return {
      status: response?.status,
      statusText: response?.statusText,
      data: response?.data
    }
  } catch (e) {
    logger.error(`Failed to insert Data. API-URL:${ORAKL_NETWORK_API_URL}, Data: ${data}`)
    throw new OraklError(OraklErrorCode.FailedInsertData)
  }
}

export async function insertAggregateData({
  aggregatorId,
  value,
  logger
}: {
  aggregatorId: bigint
  value: number
  logger: Logger
}) {
  const timestamp = new Date(Date.now()).toString()
  const data = {
    aggregatorId,
    timestamp,
    value
  }

  try {
    const url = buildUrl(ORAKL_NETWORK_API_URL, 'aggregate')
    const response = await axios.post(url, { data })
    return {
      status: response?.status,
      statusText: response?.statusText,
      data: response?.data
    }
  } catch (e) {
    logger.error(`Failed to insert Aggregated Data API-URL:${ORAKL_NETWORK_API_URL}, Data: ${data}`)
    throw new OraklError(OraklErrorCode.FailedInsertAggregatedData)
  }
}
