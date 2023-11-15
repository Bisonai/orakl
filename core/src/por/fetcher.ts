import axios from 'axios'
import { IAggregator } from '../types'
import { pipe } from '../utils'
import { insertData, loadAggregator } from './api'
import { PorError, PorErrorCode } from './errors'
import { DATA_FEED_REDUCER_MAPPING } from './reducer'

async function extractFeed(adapter) {
  const feeds = adapter.feeds.map((f) => {
    return {
      id: f.id,
      name: f.name,
      url: f.definition.url,
      headers: f.definition.headers,
      method: f.definition.method,
      reducers: f.definition.reducers
    }
  })
  return feeds[0]
}

function checkDataFormat(data) {
  if (!data) {
    throw new PorError(PorErrorCode.InvalidDataFeed)
  } else if (!Number.isInteger(data)) {
    throw new PorError(PorErrorCode.InvalidDataFeedFormat)
  }
}

function buildReducer(reducerMapping, reducers) {
  return reducers.map((r) => {
    const reducer = reducerMapping[r.function]
    if (!reducer) {
      throw new PorError(PorErrorCode.InvalidReducer)
    }
    return reducer(r?.args)
  })
}

async function fetchData(feed) {
  const rawDatum = await (await axios.get(feed.url)).data
  const reducers = await buildReducer(DATA_FEED_REDUCER_MAPPING, feed.reducers)
  const datum = pipe(...reducers)(rawDatum)
  checkDataFormat(datum)
  return datum
}

export async function fetchWithAggregator(aggregatorHash: string) {
  const aggregator: IAggregator = await loadAggregator({ aggregatorHash })
  const adapter = aggregator.adapter
  const feed = await extractFeed(adapter)
  const value = await fetchData(feed)

  await insertData({ aggregatorId: aggregator.id, feedId: feed.id, value })

  return { value: BigInt(value), aggregator }
}
