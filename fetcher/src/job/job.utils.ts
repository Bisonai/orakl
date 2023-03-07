import axios from 'axios'
import { DATA_FEED_REDUCER_MAPPING } from './job.reducer'
import { LOCAL_AGGREGATOR_FN } from './job.aggregator'
import { FetcherError, FetcherErrorCode } from './job.errors'

export function buildUrl(host: string, path: string) {
  const url = [host, path].join('/')
  return url.replace(/([^:]\/)\/+/g, '$1')
}

export async function loadAggregator(aggregatorId: string, chain: string) {
  let response = {}
  try {
    const url = buildUrl(process.env.ORAKL_NETWORK_API_URL, `aggregator/${aggregatorId}`)
    response = (await axios.get(url, { data: { chain } }))?.data
  } catch (e) {
    this.logger.error(e)
  } finally {
    return response
  }
}

/* DEPRECATED */
// export async function loadAdapters({
//   postprocess
// }: {
//   postprocess?: boolean
// }) /*: Promise<IAdapter[]>*/ {
//   const rawAdapters = [] //await getAdapters(DB, CHAIN)
//   const validatedRawAdapters = rawAdapters.map((a) => validateAdapter(JSON.parse(a.data)))
//
//   if (!postprocess) {
//     return validatedRawAdapters
//   }
//
//   const activeRawAdapters = validatedRawAdapters.filter((a) => a.active)
//   return Object.assign({}, ...activeRawAdapters.map((a) => extractFeeds(a)))
// }

/**
 * Fetch data from data sources defined in `adapter`.
 *
 * @param {} adapter Single data adapter to define which data to fetch.
 * @return {number} aggregatedresults
 */
export async function fetchData(adapter) {
  return await Promise.all(
    adapter.map(async (a) => {
      const options = {
        method: a.method,
        headers: a.headers
      }

      try {
        const rawDatum = (await axios.get(a.url, options)).data

        // FIXME Build reducers just once and use. Currently, can't
        // be passed to queue, therefore has to be recreated before
        // every fetch.
        const reducers = buildReducer(DATA_FEED_REDUCER_MAPPING, a.reducers)
        const datum = pipe(...reducers)(rawDatum)
        checkDataFormat(datum)
        return { id: a.id, value: datum }
      } catch (e) {
        console.error(`Error in ${a.name}`)
        console.error(e)
        return { id: a.id, value: undefined }
      }
    })
  )
}

// TODO define data type
export function aggregateData(data: number[]): number {
  const aggregate = LOCAL_AGGREGATOR_FN(data.map((d) => d['value']))
  return aggregate
}

function buildReducer(reducerMapping, reducers) {
  return reducers.map((r) => {
    const reducer = reducerMapping[r.function]
    if (!reducer) {
      throw new FetcherError(FetcherErrorCode.InvalidReducer)
    }
    return reducer(r?.args)
  })
}

// https://medium.com/javascript-scene/reduce-composing-software-fe22f0c39a1d
export const pipe =
  (...fns) =>
  (x) =>
    fns.reduce((v, f) => f(v), x)

function checkDataFormat(data) {
  if (!data) {
    // check if priceFeed is null, undefined, NaN, "", 0, false
    throw new FetcherError(FetcherErrorCode.InvalidDataFeed)
  } else if (!Number.isInteger(data)) {
    // check if priceFeed is not Integer
    throw new FetcherError(FetcherErrorCode.InvalidDataFeedFormat)
  }
}

function validateAdapter(adapter) /*: IAdapter*/ {
  // TODO extract properties from Interface
  const requiredProperties = ['id', 'active', 'name', 'jobType', 'decimals', 'feeds']
  // TODO show where is the error
  const hasProperty = requiredProperties.map((p) =>
    Object.prototype.hasOwnProperty.call(adapter, p)
  )
  const isValid = hasProperty.every((x) => x)

  if (isValid) {
    return adapter /*as IAdapter*/
  } else {
    throw new FetcherError(FetcherErrorCode.InvalidAdapter)
  }
}

export function extractFeeds(adapter, aggregatorId: string) {
  const adapterId = adapter.adapterId
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

  return {
    [adapterId]: {
      aggregatorId: aggregatorId,
      name: adapter.name,
      decimals: adapter.decimals,
      feeds
    }
  }
}
