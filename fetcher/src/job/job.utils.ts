import axios from 'axios'
import { DATA_FEED_REDUCER_MAPPING } from './job.reducer'
import { LOCAL_AGGREGATOR_FN } from './job.aggregator'
import { FetcherError, FetcherErrorCode } from './job.errors'
import { IAdapter, IFetchedData } from './job.types'

let latestId = 0
let proxySourceMap = {}

export function buildUrl(host: string, path: string) {
  const url = [host, path].join('/')
  return url.replace(/([^:]\/)\/+/g, '$1')
}

/**
 * Fetch data from data sources defined in `adapter`.
 *
 * @param {} adapter Single data adapter to define which data to fetch.
 * @param {} NestJs logger
 * @return {number} aggregatedresults
 */
export async function fetchData(adapter, logger) {
  const data = await Promise.allSettled(
    adapter.map(async (a) => {
      // Make request options with Proxy
      const options = {
        method: a.method,
        headers: a.header,
        proxy: {
          protocol: a.proxy.protocol,
          host: a.proxy.host,
          port: a.proxy.port
        }
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
        logger.error(`Fetching with proxy ${a.proxy.host} failed in ${a.name}`)
        logger.error(e)

        // Make request options without Proxy
        const options = {
          method: a.method,
          headers: a.headers
        }
        try {
          const rawDatum = (await axios.get(a.url, options)).data
          const reducers = buildReducer(DATA_FEED_REDUCER_MAPPING, a.reducers)
          const datum = pipe(...reducers)(rawDatum)
          checkDataFormat(datum)
          return { id: a.id, value: datum }
        } catch {
          logger.error(`Error in ${a.name}`)
          logger.error(e)
          throw e
        }
      }
    })
  )

  return data.flatMap((D) => (D.status == 'fulfilled' ? [D.value] : []))
}

export function aggregateData(data: IFetchedData[]): number {
  const aggregate = LOCAL_AGGREGATOR_FN(data.map((d) => d.value))
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

function validateAdapter(adapter): IAdapter {
  // TODO extract properties from Interface
  const requiredProperties = ['id', 'active', 'name', 'jobType', 'decimals', 'feeds']
  // TODO show where is the error
  const hasProperty = requiredProperties.map((p) =>
    Object.prototype.hasOwnProperty.call(adapter, p)
  )
  const isValid = hasProperty.every((x) => x)

  if (isValid) {
    return adapter as IAdapter
  } else {
    throw new FetcherError(FetcherErrorCode.InvalidAdapter)
  }
}

function selectProxy(url: string, proxies) {
  const source = url.split('/')[2]
  const proxySize = proxies.length

  if (!source) {
    throw new FetcherError(FetcherErrorCode.InvalidUrl)
  }
  if (proxySize == 0) {
    throw new FetcherError(FetcherErrorCode.UnexpectedNumberOfProxies)
  }

  if (source in proxySourceMap) {
    proxySourceMap[source] = (proxySourceMap[source] + 1) % proxySize
  } else {
    proxySourceMap[source] = latestId
    latestId = (latestId + 1) % proxySize
  }

  return proxies[proxySourceMap[source]]
}

export function extractFeeds(
  adapter,
  aggregatorId: bigint,
  aggregatorHash: string,
  threshold: number,
  absoluteThreshold: number,
  address: string,
  proxies
) {
  const adapterHash = adapter.adapterHash
  const feeds = adapter.feeds.map((f) => {
    return {
      id: f.id,
      name: f.name,
      url: f.definition.url,
      headers: f.definition.headers,
      method: f.definition.method,
      reducers: f.definition.reducers,
      proxy: selectProxy(f.definition.url, proxies)
    }
  })

  return {
    [adapterHash]: {
      aggregatorId: aggregatorId,
      aggregatorHash: aggregatorHash,
      name: adapter.name,
      decimals: adapter.decimals,
      threshold,
      absoluteThreshold,
      address,
      feeds
    }
  }
}

/**
 * Test whether the current submission deviates from the last
 * submission more than given threshold or absolute threshold. If yes,
 * return `true`, otherwise `false`.
 *
 * @param {number} latest submission value
 * @param {number} current submission value
 * @param {number} threshold configuration
 * @param {number} absolute threshold configuration
 * @return {boolean}
 */
export function shouldReport(
  latestSubmission: number,
  submission: number,
  decimals: number,
  threshold: number,
  absoluteThreshold: number
): boolean {
  if (latestSubmission && submission) {
    const denominator = Math.pow(10, decimals)
    const latestSubmissionReal = latestSubmission / denominator
    const submissionReal = submission / denominator

    const range = latestSubmissionReal * threshold
    const l = latestSubmissionReal - range
    const r = latestSubmissionReal + range
    return submissionReal < l || submissionReal > r
  } else if (!latestSubmission && submission) {
    // latestSubmission hit zero
    return submission > absoluteThreshold
  } else {
    // Something strange happened, don't report!
    return false
  }
}
