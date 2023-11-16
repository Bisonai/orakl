import axios from 'axios'
import { Logger } from 'pino/pino'
import { IAggregator } from '../types'
import { pipe } from '../utils'
import { insertAggregateData, insertData, loadAggregator } from './api'
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

async function fetchData(feed, logger) {
  try {
    const rawDatum = await (await axios.get(feed.url)).data
    const reducers = buildReducer(DATA_FEED_REDUCER_MAPPING, feed.reducers)
    const datum = pipe(...reducers)(rawDatum)
    checkDataFormat(datum)
    return datum
  } catch (e) {
    logger.error(`Fetching data failed for url:${feed.url}`)
    logger.error(e)
    throw e
  }
}

export async function fetchWithAggregator({
  aggregatorHash,
  logger
}: {
  aggregatorHash: string
  logger: Logger
}) {
  const aggregator: IAggregator = await loadAggregator({ aggregatorHash, logger })
  const adapter = aggregator.adapter
  const feed = await extractFeed(adapter)
  const value = await fetchData(feed, logger)

  await insertData({ aggregatorId: aggregator.id, feedId: feed.id, value, logger })
  await insertAggregateData({ aggregatorId: aggregator.id, value, logger })

  return { value: BigInt(value), aggregator }
}
