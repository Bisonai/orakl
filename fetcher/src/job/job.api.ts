import { IData } from '@bisonai/orakl-util'
import { HttpException, HttpStatus, Logger } from '@nestjs/common'
import axios from 'axios'
import { IAggregate, IAggregateById, IAggregator, IProxy, IRawData } from './job.types'
import { buildUrl } from './job.utils'

export async function loadActiveAggregators({ chain, logger }: { chain: string; logger: Logger }) {
  const AGGREGATOR_ENDPOINT = buildUrl(process.env.ORAKL_NETWORK_API_URL, 'aggregator')
  try {
    const url = new URL(AGGREGATOR_ENDPOINT)
    url.searchParams.append('active', 'true')
    url.searchParams.append('chain', chain)

    const result = (await axios.get(url.toString())).data
    return result
  } catch (e) {
    const msg = `Loading aggregators with filter active:true for chain ${chain} failed.`
    logger.error(msg)
    throw new HttpException(msg, HttpStatus.BAD_REQUEST)
  }
}

export async function loadAggregator({
  aggregatorHash,
  chain,
  logger,
}: {
  aggregatorHash: string
  chain: string
  logger: Logger
}) {
  try {
    const url = buildUrl(process.env.ORAKL_NETWORK_API_URL, `aggregator/${aggregatorHash}/${chain}`)
    const aggregator: IAggregator = (await axios.get(url))?.data
    return aggregator
  } catch (e) {
    const msg = `Loading aggregator with hash ${aggregatorHash} for chain ${chain} failed.`
    logger.error(msg)
    throw new HttpException(msg, HttpStatus.BAD_REQUEST)
  }
}

export async function insertMultipleData({
  aggregatorId,
  timestamp,
  data,
}: {
  aggregatorId: string
  timestamp: string
  data: IRawData[]
}) {
  const _data: IData[] = data.map((d) => {
    return {
      aggregatorId: aggregatorId,
      feedId: d.id,
      timestamp: timestamp,
      value: d.value,
    }
  })

  const url = buildUrl(process.env.ORAKL_NETWORK_API_URL, 'data')
  const response = await axios.post(url, { data: _data })
  return {
    status: response?.status,
    statusText: response?.statusText,
    data: response?.data,
  }
}

export async function insertAggregateData({
  aggregatorId,
  timestamp,
  value,
}: {
  aggregatorId: string
  timestamp: string
  value: number
}) {
  const url = buildUrl(process.env.ORAKL_NETWORK_API_URL, 'aggregate')
  const response = await axios.post(url, { data: { aggregatorId, timestamp, value } })
  return {
    status: response?.status,
    statusText: response?.statusText,
    data: response?.data,
  }
}

async function updateAggregator(aggregatorHash: string, chain: string, active: boolean) {
  const url = buildUrl(process.env.ORAKL_NETWORK_API_URL, `aggregator/${aggregatorHash}`)
  const response = await axios.patch(url, { data: { active, chain } })
  return response?.data
}

export async function activateAggregator(aggregatorHash: string, chain: string) {
  return await updateAggregator(aggregatorHash, chain, true)
}

export async function deactivateAggregator(aggregatorHash: string, chain: string) {
  return await updateAggregator(aggregatorHash, chain, false)
}

export async function fetchDataFeed({
  aggregatorHash,
  logger,
}: {
  aggregatorHash: string
  logger: Logger
}): Promise<IAggregate> {
  try {
    const url = buildUrl(process.env.ORAKL_NETWORK_API_URL, `aggregate/${aggregatorHash}/latest`)
    return (await axios.get(url))?.data
  } catch (e) {
    logger.error(e)
  }
}

export async function fetchDataFeedByAggregatorId({
  aggregatorId,
  logger,
}: {
  aggregatorId: bigint
  logger: Logger
}): Promise<IAggregateById> {
  try {
    const url = buildUrl(process.env.ORAKL_NETWORK_API_URL, `aggregate/id/${aggregatorId}/latest`)
    return (await axios.get(url))?.data
  } catch (e) {
    logger.error(e)
  }
}

export async function loadProxies({ logger }: { logger: Logger }): Promise<IProxy[]> {
  try {
    const url = buildUrl(process.env.ORAKL_NETWORK_API_URL, 'proxy')
    const proxies: IProxy[] = (await axios.get(url))?.data
    return proxies
  } catch (e) {
    const msg = 'Loading proxies failed.'
    logger.error(msg)
    throw new HttpException(msg, HttpStatus.BAD_REQUEST)
  }
}
