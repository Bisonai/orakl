import axios from 'axios'
import { pipe } from '../utils'
import { loadAggregator } from './api'
import { PorError, PorErrorCode } from './errors'
import { DATA_FEED_REDUCER_MAPPING } from './reducer'

async function extractFeeds(adapter) {
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
  return feeds
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
  try {
    const rawDatum = (await axios.get(feed.url)).data
    const reducers = buildReducer(DATA_FEED_REDUCER_MAPPING, feed.reducers)
    const datum = pipe(...reducers)(rawDatum)
    checkDataFormat(datum)
    return datum
  } catch (e) {
    throw e
  }
}

export async function fetchWithAggregator(aggregatorHash: string) {
  try {
    const aggregator = await loadAggregator({ aggregatorHash })
    const adapter = aggregator.adapter
    const feeds = await extractFeeds(adapter)
    return await fetchData(feeds[0])
  } catch (e) {
    throw e
  }
}
