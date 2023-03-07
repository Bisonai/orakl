import axios from 'axios'
import { buildUrl } from './job.utils'

export async function insertMultipleData({
  aggregatorId,
  timestamp,
  data
}: {
  aggregatorId: string
  timestamp: string
  data: any[]
}) {
  const formattedData = data.map((d) => {
    return {
      aggregator: aggregatorId,
      timestamp,
      value: d.value,
      feed: d.id
    }
  })

  const ORAKL_NETWORK_API_DATA = buildUrl(process.env.ORAKL_NETWORK_API_URL, 'data')
  const response = await axios.post(ORAKL_NETWORK_API_DATA, { data: formattedData })
  console.log(response.data)
}
