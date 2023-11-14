import axios from 'axios'
import { OraklError, OraklErrorCode } from '../errors'
import { CHAIN, ORAKL_NETWORK_API_URL } from '../settings'
import { IAggregator } from '../types'
import { buildUrl } from '../utils'
import { IData } from './types'

export async function loadAggregator({ aggregatorHash }: { aggregatorHash: string }) {
  const chain = CHAIN
  try {
    const url = buildUrl(ORAKL_NETWORK_API_URL, `aggregator/${aggregatorHash}/${chain}`)
    const aggregator: IAggregator = (await axios.get(url))?.data
    return aggregator
  } catch (e) {
    throw new OraklError(OraklErrorCode.GetListenerRequestFailed)
  }
}

export async function insertData({
  aggregatorId,
  feedId,
  value
}: {
  aggregatorId: bigint
  feedId: bigint
  value: number
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

  const url = buildUrl(ORAKL_NETWORK_API_URL, 'data')
  const response = await axios.post(url, { data })

  return {
    status: response?.status,
    statusText: response?.statusText,
    data: response?.data
  }
}
