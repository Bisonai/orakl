import axios from 'axios'
import { DATA_FEED_REDUCER_MAPPING } from './job.reducer'
import { LOCAL_AGGREGATOR_FN } from './job.aggregator'
import { FetcherError, FetcherErrorCode } from './job.errors'
import { IAdapter, IFetchedData, IProxy } from './job.types'
import { Logger } from '@nestjs/common'
import { ethers } from 'ethers'
import { Aggregator__factory } from '@bisonai/orakl-contracts'
import { PROVIDER } from 'src/settings'

export function buildUrl(host: string, path: string) {
  const url = [host, path].join('/')
  return url.replace(/([^:]\/)\/+/g, '$1')
}

function isProxyDefined(adapter) {
  return (
    adapter.proxy !== undefined &&
    adapter.proxy.protocol !== undefined &&
    adapter.proxy.host !== undefined &&
    adapter.proxy.port !== undefined
  )
}

const INVALID_DATA = -1
async function fetchCall(url: string, options, logger) {
  try {
    return (await axios.get(url, options)).data
  } catch (e) {
    logger.error(`Error in fetching data from ${url}: ${e.message}`)
    logger.error(e)
    return INVALID_DATA
  }
}

async function fetchRawDataWithProxy(adapter, logger) {
  return fetchCall(
    adapter.url,
    {
      method: adapter.method,
      headers: adapter.headers,
      proxy: {
        protocol: adapter.proxy.protocol,
        host: adapter.proxy.host,
        port: adapter.proxy.port
      }
    },
    logger
  )
}

async function fetchRawDataWithoutProxy(adapter, logger) {
  return fetchCall(
    adapter.url,
    {
      method: adapter.method,
      headers: adapter.headers
    },
    logger
  )
}

/**
 * Fetch data from data sources defined in `adapter`.
 *
 * @param {} adapter Single data adapter to define which data to fetch.
 * @param {} NestJs logger
 * @return {number} aggregatedresults
 */
export async function fetchData(adapterList, logger) {
  const data = await Promise.allSettled(
    adapterList.map(async (adapter) => {
      let rawDatum = INVALID_DATA
      if (isProxyDefined(adapter)) {
        rawDatum = await fetchRawDataWithProxy(adapter, logger)
      }
      if (rawDatum === INVALID_DATA) {
        rawDatum = await fetchRawDataWithoutProxy(adapter, logger)
        if (rawDatum === INVALID_DATA) {
          throw new Error(`Error in fetching data`)
        }
      }
      try {
        // FIXME Build reducers just once and use. Currently, can't
        // be passed to queue, therefore has to be recreated before
        // every fetch.
        const reducers = buildReducer(DATA_FEED_REDUCER_MAPPING, adapter.reducers)
        const datum = pipe(...reducers)(rawDatum)
        checkDataFormat(datum)
        return { id: adapter.id, value: datum }
      } catch (e) {
        logger.error(`Fetching with proxy ${adapter.proxy.host} failed in ${adapter.name}`)
        logger.error(e)
        throw e
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

function selectProxyFn(proxies: IProxy[]) {
  let latestId = 0
  const proxySourceMap: { [host: string]: number } = {}

  function wrapper(url: string): IProxy {
    const source = new URL(url).host
    const proxySize = proxies.length
    if (proxySize == 0) {
      return { protocol: undefined, host: undefined, port: undefined }
    }

    if (source in proxySourceMap) {
      proxySourceMap[source] = (proxySourceMap[source] + 1) % proxySize
    } else {
      proxySourceMap[source] = latestId
      latestId = (latestId + 1) % proxySize
    }
    return proxies[proxySourceMap[source]]
  }
  return wrapper
}

export function extractFeeds(
  adapter,
  aggregatorId: bigint,
  aggregatorHash: string,
  threshold: number,
  absoluteThreshold: number,
  address: string,
  proxies: IProxy[],
  logger: Logger
) {
  const adapterHash = adapter.adapterHash
  const proxySelector = selectProxyFn(proxies)
  const feeds = adapter.feeds.map((f) => {
    let proxy: IProxy
    try {
      if (!f.definition.location) {
        proxy = proxySelector(f.definition.url)
      } else {
        const availableProxies = proxies.filter(
          (item) => item.location && item.location === f.definition.location
        )
        if (availableProxies.length == 0) {
          throw `no proxies available for location:${f.definition.location}`
        }
        const randomIndex = Math.floor(Math.random() * availableProxies.length)
        proxy = availableProxies[randomIndex]
      }
    } catch (e) {
      logger.error('Assigning proxy has failed')
      logger.error(e)
    }

    return {
      id: f.id,
      name: f.name,
      url: f.definition.url,
      headers: f.definition.headers,
      method: f.definition.method,
      reducers: f.definition.reducers,
      proxy
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
 * @param {number} aggregatorAddress - Aggregator contract address
 * @param {number} currentSubmission - Submission value
 * @param {number} threshold - Threshold configuration
 * @param {number} absoluteThreshold - AbsoluteThreshold configuration
 * @return {boolean} Result in boolean, True or False
 */

export async function shouldReport(
  aggregatorAddress: string,
  currentSubmission: number,
  threshold: number,
  absoluteThreshold: number
): Promise<boolean> {
  const contract = new ethers.Contract(aggregatorAddress, Aggregator__factory.abi, PROVIDER)
  const latestSubmission = Number((await contract.latestRoundData()).answer)

  if (latestSubmission && currentSubmission && threshold) {
    // Check deviation threshold
    const range = latestSubmission * threshold
    const l = latestSubmission - range
    const r = latestSubmission + range
    return currentSubmission < l || currentSubmission > r
  } else if (!latestSubmission && currentSubmission) {
    // latestSubmission hit zero
    return currentSubmission > absoluteThreshold
  } else {
    // Something strange happened, don't report!
    return false
  }
}
