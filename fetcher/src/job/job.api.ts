import axios from 'axios'
import { buildUrl } from './job.utils'

export async function loadAggregator(aggregatorHash: string, chain: string) {
  let response = {}
  try {
    const url = buildUrl(process.env.ORAKL_NETWORK_API_URL, `aggregator/${aggregatorHash}`)
    response = (await axios.get(url, { data: { chain } }))?.data
  } catch (e) {
    this.logger.error(e)
  } finally {
    return response
  }
}

export async function insertMultipleData({
  aggregatorId,
  timestamp,
  data
}: {
  aggregatorId: string
  timestamp: string
  data: any[]
}) {
  const _data = data.map((d) => {
    return {
      aggregatorId: aggregatorId,
      feedId: d.id,
      timestamp: timestamp,
      value: d.value
    }
  })

  const url = buildUrl(process.env.ORAKL_NETWORK_API_URL, 'data')
  const response = await axios.post(url, { data: _data })
  return {
    status: response?.status,
    statusText: response?.statusText,
    data: response?.data
  }
}

export async function insertAggregateData({
  aggregatorId,
  timestamp,
  value
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
    data: response?.data
  }
}

export async function updateAggregator(aggregatorHash: string, chain: string, active: boolean) {
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
