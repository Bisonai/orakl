import axios from 'axios'
import { OraklError, OraklErrorCode } from '../errors'
import { CHAIN, ORAKL_NETWORK_API_URL } from '../settings'
import { IAggregator } from '../types'
import { buildUrl, pipe } from '../utils'
import { DATA_FEED_REDUCER_MAPPING } from './reducer'

const aggregatorHash = '0x952f883b8d2fd47a790307cb569118a215ea45eb861cefd4ed3b83ae7550f8e8'

async function loadAggregator({ aggregatorHash }: { aggregatorHash: string }) {
  const chain = CHAIN
  try {
    const url = buildUrl(ORAKL_NETWORK_API_URL, `aggregator/${aggregatorHash}/${chain}`)
    const aggregator: IAggregator = (await axios.get(url))?.data
    console.log('Aggregator:', aggregator)
    return aggregator
  } catch (e) {
    throw new OraklError(OraklErrorCode.GetListenerRequestFailed)
  }
}

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
    // check if priceFeed is null, undefined, NaN, "", 0, false
    // throw new FetcherError(FetcherErrorCode.InvalidDataFeed)
  } else if (!Number.isInteger(data)) {
    // check if priceFeed is not Integer
    // throw new FetcherError(FetcherErrorCode.InvalidDataFeedFormat)
  }
}

function buildReducer(reducerMapping, reducers) {
  return reducers.map((r) => {
    const reducer = reducerMapping[r.function]
    if (!reducer) {
      // trowError
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
    console.log('DATUM:', datum)
    return { id: feed.id, value: rawDatum }
  } catch (e) {
    return e
  }

  try {
    // FIXME Build reducers just once and use. Currently, can't
    // be passed to queue, therefore has to be recreated before
    // every fetch.
    // const reducers = buildReducer(DATA_FEED_REDUCER_MAPPING, adapter.reducers)
    // const datum = pipe(...reducers)(rawDatum)
    // checkDataFormat(datum)
  } catch (e) {
    throw e
  }
}

async function fetchAggregator(aggregatorHash: string) {
  const aggregator = await loadAggregator({ aggregatorHash })
  const adapter = aggregator.adapter
  const feeds = await extractFeeds(adapter)
  console.log(feeds)

  const data = await fetchData(feeds[0])
  // const aggregate = aggregateData(data)
}

// /**
//  * Fetch single reporter given its ID from the Orakl Network API.
//  *
//  * @param {string} reporter ID
//  * @param {pino.Logger} logger
//  * @return {IReporterConfig} reporter configuration
//  * @exception {GetReporterRequestFailed}
//  */
// export async function getReporter({
//   id,
//   logger
// }: {
//   id: string
//   logger?: Logger
// }): Promise<IReporterConfig> {
//   //   try {
//   //     const endpoint = buildUrl(ORAKL_NETWORK_API_URL, `reporter/${id}`)
//   //     return (await axios.get(endpoint))?.data
//   //   } catch (e) {
//   //     logger?.error({ name: 'getReporter', file: FILE_NAME, ...e }, 'error')
//   //     throw new OraklError(OraklErrorCode.GetReporterRequestFailed)
//   //   }
// }

const main = async () => {
  console.log('TO Main')
  const data = fetchAggregator(aggregatorHash)
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
