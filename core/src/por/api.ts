import axios from 'axios'
import { OraklError, OraklErrorCode } from '../errors'
import { CHAIN, ORAKL_NETWORK_API_URL } from '../settings'
import { IAggregator } from '../types'
import { buildUrl } from '../utils'

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
