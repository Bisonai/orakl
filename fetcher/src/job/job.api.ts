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
      aggregatorId,
      timestamp,
      value: d.value,
      feedId: d.id
    }
  })

  const url = buildUrl(process.env.ORAKL_NETWORK_API_URL, 'data')
  const response = await axios.post(url, { data: _data })
  console.log(response.data)
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
